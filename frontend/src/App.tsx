import { useEffect, useState } from 'react';
import { BrowserRouter as Router, Routes, Route, Link } from 'react-router-dom';
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
      <nav style={{ margin: '10px' }}>
        <Link to="/" style={{ marginRight: '10px' }}>home</Link>
        {!isAuth ? (<>
        <Link to="/login" style={{ marginRight: '10px' }}>login</Link>
        <Link to="/register" style={{ marginRight: '10px' }}>register</Link>
        </>):(<></>)
        }
        {isAuth ?(
        <>
        <Link to="/chatroom" style={{ marginRight: '10px' }}>chatroom</Link>
        <Link to="/searchchatroom" style={{ marginRight: '10px' }}>search chatroom</Link>
        </>):(<></>)
        }
      </nav>

      <Routes>
        <Route path="/" element={<HomePage setIsAuth={setIsAuth} />} />
        <Route element={<PublicRoute isAuthenticated={isAuth} />}>
            <Route path="/login" element={<LoginForm setIsAuth={setIsAuth} />} />
            <Route path="/register" element={<RegisterForm />} />
        </Route>
        <Route path="/chatroom" element={<ChatRoom />} />
        <Route path="/searchchatroom" element={<SearchPage />} />
      </Routes>

    </Router>
  //  </UserProvider>
  );
}

export default App;
