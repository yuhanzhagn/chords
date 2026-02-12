import { useEffect, useState } from 'react';
//import { useUser } from '../context/UserProvider';
import { useNavigate } from 'react-router-dom';


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
    <div style={{ padding: '20px' }}>
      <h2>Welcome to the Home Page</h2>

      {token ? (
        <>
          <p>You are logged in! ?</p>
          <button onClick={handleLogout}>Logout</button>
        </>
      ) : (
        <p>Please <a href="/login">log in</a> or <a href="/register">register</a>.</p>
      )}
    </div>
  );
}

export default HomePage;
