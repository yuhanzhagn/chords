import { Navigate, Outlet } from "react-router-dom";

interface PublicRouteProps {
  isAuthenticated: boolean;
}

// Component to guard routes that should NOT be accessed by logged-in users
const PublicRoute = ({ isAuthenticated }: PublicRouteProps) => {
  return isAuthenticated ? <Navigate to="../"  /> : <Outlet />;
};

export default PublicRoute;
