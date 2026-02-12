import { useState } from 'react';
import type { FormEvent } from 'react';
import { useNavigate } from 'react-router-dom';

interface LoginFormProps {
  setIsAuth: (isAuth: boolean) => void;
}

interface LoginResponse {
  token: string;
  sessionID: string;
  id: string | number;
}

function LoginForm({ setIsAuth }: LoginFormProps) {
  const [username, setUsername] = useState('');
  const [password, setPassword] = useState('');
  const [error, setError] = useState('');
  const [loading, setLoading] = useState(false);
  const navigate = useNavigate();

  const handleSubmit = async (e: FormEvent<HTMLFormElement>) => {
    e.preventDefault();
    setError('');
    setLoading(true);

    try {
      const apiUrl = process.env.REACT_APP_URL;
      if (!apiUrl) {
        throw new Error('Missing REACT_APP_URL environment variable');
      }

      const response = await fetch(`http://${apiUrl}/auth/login`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ username, password }),
      });

      if (!response.ok) {
        throw new Error('Invalid credentials');
      }

      const data: LoginResponse = await response.json();
      const token = data.token;
      const sessionID = data.sessionID;
      const userInfo = {
        id: data.id,
        username,
      };

      if (!token) throw new Error('No token received from server');
      // console.log(token);
      // ? Store JWT in localStorage
      localStorage.setItem('jwt', token);
      localStorage.setItem('user', JSON.stringify(userInfo));
      localStorage.setItem('sessionID', sessionID);

      
      setIsAuth(true);
      navigate('../chatroom');
      console.log('Logged in successfully. JWT stored.');
    } catch (err: unknown) {
      setError(err instanceof Error ? err.message : 'Login failed');
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
