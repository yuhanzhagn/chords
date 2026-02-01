import { Navigate, Outlet } from "react-router-dom";
import HomePage from "./homepage/HomePage";

// Component to guard routes that should NOT be accessed by logged-in users
const PublicRoute = ({ isAuthenticated }) => {
  return isAuthenticated ? <Navigate to="../"  /> : <Outlet />;
};

export default PublicRoute;

