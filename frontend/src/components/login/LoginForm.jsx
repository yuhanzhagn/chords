import React, { useState } from 'react';
import {useUser} from '../context/UserProvider';
import {useNavigate} from 'react-router-dom';

function LoginForm({ setIsAuth }) {
  const [username, setUsername] = useState('');
  const [password, setPassword] = useState('');
  const [error, setError] = useState('');
  const [loading, setLoading] = useState(false);
  const navigate = useNavigate();

  const handleSubmit = async (e) => {
    e.preventDefault();
    setError('');
    setLoading(true);

    try {
      var uname = username;
      const response = await fetch(`http://${process.env.REACT_APP_URL}/auth/login`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ username, password }),
      });

      if (!response.ok) {
        throw new Error('Invalid credentials');
      }

      const data = await response.json();
      const token = data.token;
      const sessionID = data.sessionID;
      const userInfo = {
        id: data.id,
        username: uname,
        };

      if (!token) throw new Error('No token received from server');
      // console.log(token);
      // ? Store JWT in localStorage
      localStorage.setItem('jwt', token);
      localStorage.setItem('user', JSON.stringify(userInfo));
      localStorage.setItem('sessionID', sessionID);

      // Optionally pass user data to parent
      //onLogin?.(data);
      setIsAuth(true);
      navigate('../chatroom');
      console.log('Logged in successfully. JWT stored.');
    } catch (err) {
      setError(err.message);
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="login-container">
      <h2>Login</h2>
      <form onSubmit={handleSubmit}>
    <label>
        Username:
        <input
          type="text"
          value={username}
          onChange={(e) => setUsername(e.target.value)}
          required
        />
      </label>
        <label>
          Password:
          <input
            type="password"
            value={password}
            onChange={(e) => setPassword(e.target.value)}
            required
          />
        </label>

        {error && <p className="error">{error}</p>}
        <button type="submit" disabled={loading}>
          {loading ? 'Logging in...' : 'Log In'}
        </button>
      </form>
    </div>
  );
}

export default LoginForm;

