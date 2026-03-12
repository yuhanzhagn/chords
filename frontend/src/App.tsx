import { useEffect, useState } from 'react';
import { BrowserRouter as Router, Routes, Route, Link } from 'react-router-dom';
import './App.css';
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
      <div className="app-shell">
        <header className="topbar">
          <div className="brand">
            <div className="brand-mark" aria-hidden />
            <div className="brand-text">
              <div className="brand-title">GoChatroom</div>
              <div className="brand-subtitle">Realtime chat, clean and fast</div>
            </div>
          </div>
          <nav className="nav-links">
            <Link to="/" className="nav-link">Home</Link>
            {!isAuth ? (
              <>
                <Link to="/login" className="nav-link">Login</Link>
                <Link to="/register" className="nav-link nav-link--primary">Register</Link>
              </>
            ) : null}
            {isAuth ? (
              <>
                <Link to="/chatroom" className="nav-link">Chatroom</Link>
                <Link to="/searchchatroom" className="nav-link">Search</Link>
              </>
            ) : null}
          </nav>
        </header>

        <main className="app-main">
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
