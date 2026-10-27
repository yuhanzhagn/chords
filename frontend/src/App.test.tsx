import { render, screen } from '@testing-library/react';
import App from './App';

const originalFetch = global.fetch;

afterEach(() => {
  global.fetch = originalFetch;
  localStorage.clear();
});

test('renders the home link without a running backend', async () => {
  // App checks the current token on mount; keep this smoke test offline.
  global.fetch = jest.fn().mockResolvedValue({ ok: false } as Response);
  render(<App />);
  expect(await screen.findByRole('link', { name: /^home$/i })).toBeInTheDocument();
});
