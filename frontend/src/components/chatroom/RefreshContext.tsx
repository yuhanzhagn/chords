import { createContext } from 'react';
import './chat.css'

export const RefreshContext = createContext<() => void>(() => {});
