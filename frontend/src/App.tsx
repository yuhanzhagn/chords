import { useEffect, useState } from 'react';
import { BrowserRouter as Router, Routes, Route, Link } from 'react-router-dom';
import { Badge } from './components/ui/badge';
import { Button } from './components/ui/button';
import HomePage from './components/homepage/HomePage';
import LoginForm from './components/login/LoginForm';
import RegisterForm from './components/register/RegisterForm';
import ChatRoom from './components/chatroom/ChatRoom';
import SearchPage from './components/searchchatroom/SearchPage';
import PublicRoute from './components/PublicRoute';
//import { UserProvider } from './components/context/UserProvider'; 

function App() {
  const ipaddr = `${process.env.REACT_APP_URL}`;
  const jwttoken = localStorage.getItem('jwt');
  const [isAuth, setIsAuth] = useState(Boolean(jwttoken));

  useEffect(() => {
    fetch(`http://${ipaddr}/chatrooms`, {
        method: "GET",
        headers: {
            "Content-Type": "application/json",
            "Authorization": `Bearer ${jwttoken}`, // <-- JWT here
            },
      }).then((res) => {
        if (res.ok) {
          setIsAuth(true);
          console.log('Token is valid');
        } else {
          setIsAuth(false);
        }
      });
  }, [ipaddr, jwttoken]);

  return (
  //  <UserProvider>
    <Router>
      <div className="min-h-screen">
        <header className="sticky top-0 z-10 border-b border-border/70 bg-background/80 backdrop-blur">
          <div className="mx-auto flex w-full max-w-6xl items-center justify-between gap-6 px-6 py-6">
            <div className="flex items-center gap-4">
              <div
                className="h-12 w-12 rounded-2xl bg-gradient-to-br from-primary via-amber-400 to-accent shadow-soft"
                aria-hidden
              />
              <div>
                <div className="text-lg font-semibold tracking-tight">GoChatroom</div>
                <div className="text-xs text-muted-foreground">
                  Realtime chat, clean and fast
                </div>
              </div>
            </div>
            <nav className="flex flex-wrap items-center gap-2">
              <Button asChild variant="ghost" size="sm">
                <Link to="/">Home</Link>
              </Button>
              {!isAuth ? (
                <>
                  <Button asChild variant="secondary" size="sm">
                    <Link to="/login">Login</Link>
                  </Button>
                  <Button asChild size="sm">
                    <Link to="/register">Register</Link>
                  </Button>
                </>
              ) : null}
              {isAuth ? (
                <>
                  <Button asChild variant="secondary" size="sm">
                    <Link to="/chatroom">Chatroom</Link>
                  </Button>
                  <Button asChild variant="ghost" size="sm">
                    <Link to="/searchchatroom">Search</Link>
                  </Button>
                  <Badge variant="accent" className="ml-2 hidden md:inline-flex">
                    Signed in
                  </Badge>
                </>
              ) : null}
            </nav>
          </div>
        </header>

        <main className="mx-auto w-full max-w-6xl px-6 py-10">
          <Routes>
            <Route path="/" element={<HomePage setIsAuth={setIsAuth} />} />
            <Route element={<PublicRoute isAuthenticated={isAuth} />}>
              <Route path="/login" element={<LoginForm setIsAuth={setIsAuth} />} />
              <Route path="/register" element={<RegisterForm />} />
            </Route>
            <Route path="/chatroom" element={<ChatRoom />} />
            <Route path="/searchchatroom" element={<SearchPage />} />
          </Routes>
        </main>
      </div>

    </Router>
  //  </UserProvider>
  );
}

export default App;
