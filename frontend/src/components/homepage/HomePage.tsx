import { useEffect, useState } from 'react';
//import { useUser } from '../context/UserProvider';
import { Link, useNavigate } from 'react-router-dom';
import { Badge } from '../ui/badge';
import { Button } from '../ui/button';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '../ui/card';


interface HomePageProps {
  setIsAuth: (isAuth: boolean) => void;
}

function HomePage({ setIsAuth }: HomePageProps) {
  const [token, setToken] = useState<string | null>(null);
  const navigate = useNavigate();
 // const { removeUserInfo } = useUser();

  useEffect(() => {
    const storedToken = localStorage.getItem('jwt');
    setToken(storedToken);
  }, []);

/*  const handleLogout = () => {
    localStorage.removeItem('jwt');
    setToken(null);
   // removeUserInfo(); 
  };
*/

const handleLogout = async () => {
  const token = localStorage.getItem('jwt');
  const apiUrl = process.env.REACT_APP_URL;

  // Optional: notify backend to block token
  if (token && apiUrl) {
    try {
      await fetch(`http://${apiUrl}/auth/logout`, {
        method: 'POST',
        headers: {
          Authorization: `Bearer ${token}`
        },
        body: JSON.stringify({
            sessionID: localStorage.getItem("sessionID")
        })
      });
        setIsAuth(false);
    } catch (err: unknown) {
      console.error('Logout request failed:', err);
    }
  }

  // Clear client-side state
  localStorage.removeItem('jwt');
  setToken(null);

  // If you store user info
  // removeUserInfo();

  // Redirect to login page
  navigate('/login');
};


  return (
    <Card className="border-border/60 bg-card/90">
      <CardContent className="grid gap-8 p-8 md:grid-cols-[1.2fr_0.8fr] md:items-center">
        <div className="space-y-6">
          <Badge variant="accent">
            {token ? "Authenticated session" : "Guest mode"}
          </Badge>
          <div className="space-y-3">
            <h2 className="text-3xl font-semibold tracking-tight md:text-4xl">
              Welcome to your realtime chat hub.
            </h2>
            <p className="text-muted-foreground">
              Jump into live rooms, search active spaces, and keep conversations moving
              with low-latency messaging. This UI is designed to stay focused on what matters:
              the chat.
            </p>
          </div>
          <div className="flex flex-wrap gap-3">
            {token ? (
              <>
                <Button asChild>
                  <Link to="/chatroom">Open Chatroom</Link>
                </Button>
                <Button asChild variant="secondary">
                  <Link to="/searchchatroom">Search Rooms</Link>
                </Button>
                <Button variant="outline" onClick={handleLogout}>
                  Logout
                </Button>
              </>
            ) : (
              <>
                <Button asChild>
                  <Link to="/login">Login</Link>
                </Button>
                <Button asChild variant="secondary">
                  <Link to="/register">Create Account</Link>
                </Button>
              </>
            )}
          </div>
        </div>
        <Card className="border-border/70 bg-secondary/40 shadow-none">
          <CardHeader>
            <CardTitle className="text-lg">Quick tips</CardTitle>
            <CardDescription>Stay focused and keep rooms tidy.</CardDescription>
          </CardHeader>
          <CardContent className="space-y-2 text-sm text-muted-foreground">
            <p>Keep one tab per room for better focus.</p>
            <p>Search rooms to discover new conversations.</p>
            <p>Reconnects are smoothed to avoid spikes.</p>
          </CardContent>
        </Card>
      </CardContent>
    </Card>
  );
}

export default HomePage;
