import React, { createContext, useContext, useState, useEffect, ReactNode } from 'react';
import { authAPI, sessionsAPI } from '../services/api';
import type { User } from '../types';

interface AuthContextType {
  user: User | null;
  loading: boolean;
  login: (email: string, password: string) => Promise<void>;
  register: (email: string, username: string, password: string) => Promise<void>;
  logout: () => Promise<void>;
  isAuthenticated: boolean;
}

const AuthContext = createContext<AuthContextType | undefined>(undefined);

export const AuthProvider: React.FC<{ children: ReactNode }> = ({ children }) => {
  const [user, setUser] = useState<User | null>(null);
  const [loading, setLoading] = useState(true);
  const [sessionVerified, setSessionVerified] = useState(false);

  useEffect(() => {
    const verifySession = async () => {
      try {
        await sessionsAPI.getSessions();
        setSessionVerified(true);
      } catch (error) {
        setUser(null);
        setSessionVerified(false);
      } finally {
        setLoading(false);
      }
    };

    verifySession();
  }, []);

  const login = async (email: string, password: string) => {
    setUser(null);
    setSessionVerified(false);
    
    const userData = await authAPI.login(email, password);
    setUser(userData);
    setSessionVerified(true);
    setLoading(false);
  };

  const register = async (email: string, username: string, password: string) => {
    setUser(null);
    setSessionVerified(false);
    
    const userData = await authAPI.register(email, username, password);
    setUser(userData);
    setSessionVerified(true);
    setLoading(false);
  };

  const logout = async () => {
    try {
      await authAPI.logout();
    } catch (error) {
      console.error('Logout error:', error);
    } finally {
      setUser(null);
      setSessionVerified(false);
      setLoading(false);
    }
  };

  return (
    <AuthContext.Provider
      value={{
        user,
        loading,
        login,
        register,
        logout,
        isAuthenticated: !!user || sessionVerified,
      }}
    >
      {children}
    </AuthContext.Provider>
  );
};

export const useAuth = () => {
  const context = useContext(AuthContext);
  if (context === undefined) {
    throw new Error('useAuth must be used within an AuthProvider');
  }
  return context;
};



