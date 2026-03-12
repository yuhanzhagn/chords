import { useEffect, useState } from 'react';
//import { useUser } from '../context/UserProvider';
import { Link, useNavigate } from 'react-router-dom';


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
    <section className="page-card">
      <div className="hero">
        <div>
          <div className="status-pill">
            {token ? "Authenticated session" : "Guest mode"}
          </div>
          <h2 className="hero-title">Welcome to your realtime chat hub.</h2>
          <p className="hero-text">
            Jump into live rooms, search active spaces, and keep conversations moving
            with low-latency messaging. This UI is designed to stay focused on what matters:
            the chat.
          </p>
          <div className="hero-actions">
            {token ? (
              <>
                <Link to="/chatroom" className="button">Open Chatroom</Link>
                <Link to="/searchchatroom" className="button secondary">Search Rooms</Link>
                <button className="button secondary" onClick={handleLogout}>Logout</button>
              </>
            ) : (
              <>
                <Link to="/login" className="button">Login</Link>
                <Link to="/register" className="button secondary">Create Account</Link>
              </>
            )}
          </div>
        </div>
        <div className="page-card" style={{ background: "var(--surface-2)" }}>
          <h3 style={{ marginTop: 0 }}>Quick tips</h3>
          <ul style={{ margin: 0, paddingLeft: "18px", color: "var(--muted)" }}>
            <li>Keep one tab per room for better focus.</li>
            <li>Search rooms to discover new conversations.</li>
            <li>Reconnects are smoothed to avoid spikes.</li>
          </ul>
        </div>
      </div>
    </section>
  );
}

export default HomePage;
