import { useState } from 'react';
import type { FormEvent } from 'react';
import { useNavigate } from 'react-router-dom';
import { Button } from '../ui/button';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '../ui/card';
import { Input } from '../ui/input';
import { Label } from '../ui/label';

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
    <div className="mx-auto w-full max-w-md">
      <Card className="border-border/70 bg-card/90">
        <CardHeader>
          <CardTitle>Login</CardTitle>
          <CardDescription>Pick up where your chats left off.</CardDescription>
        </CardHeader>
        <CardContent>
          <form onSubmit={handleSubmit} className="space-y-4">
            <div className="space-y-2">
              <Label htmlFor="login-username">Username</Label>
              <Input
                id="login-username"
                type="text"
                value={username}
                onChange={(e) => setUsername(e.target.value)}
                required
              />
            </div>
            <div className="space-y-2">
              <Label htmlFor="login-password">Password</Label>
              <Input
                id="login-password"
                type="password"
                value={password}
                onChange={(e) => setPassword(e.target.value)}
                required
              />
            </div>

            {error && (
              <div className="rounded-lg border border-destructive/40 bg-destructive/10 px-3 py-2 text-sm text-destructive">
                {error}
              </div>
            )}
            <Button type="submit" className="w-full" disabled={loading}>
              {loading ? 'Logging in...' : 'Log In'}
            </Button>
          </form>
        </CardContent>
      </Card>
    </div>
  );
}

export default LoginForm;
