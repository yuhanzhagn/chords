import { createContext, useState, useContext, useRef } from 'react';
import type { MutableRefObject, ReactNode } from 'react';

interface UserInfo {
  [key: string]: unknown;
}

interface UserContextValue {
  userRef: MutableRefObject<UserInfo | null>;
  updateUserInfo: (userData: UserInfo) => void;
  removeUserInfo: () => void;
}

const UserContext = createContext<UserContextValue | null>(null);

// Export a hook for convenience
export const useUser = () => {
  return useContext(UserContext);
};

interface UserProviderProps {
  children: ReactNode;
}

export const UserProvider = ({ children }: UserProviderProps) => {
  const [user, setUser] = useState<UserInfo | null>({});
  const userRef = useRef(user);

  const updateUserInfo = (userData: UserInfo) => {
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
