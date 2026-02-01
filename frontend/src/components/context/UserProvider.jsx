import { createContext, useState, useContext, useRef } from "react";

// Create the context
const UserContext = createContext(null);

// Export a hook for convenience
export const useUser = () => {
    return useContext(UserContext);
};

// Provider component
export const UserProvider = ({ children }) => {
  const [user, setUser] = useState({});
  const userRef = useRef(user);

  const updateUserInfo = (userData) => {
    console.log(userData);
    userRef.current = userData;
    setUser(userData);
  };

  const removeUserInfo = () => {
    setUser(null);
  };

  return (
    <UserContext.Provider value={{ userRef, updateUserInfo, removeUserInfo }}>
      {children}
    </UserContext.Provider>
  );
};

