import React, { createContext, useContext, useState, useEffect, ReactNode } from 'react';
import { authAPI } from '../services/api';
import { wsService } from '../services/websocket';
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
        const userData = await authAPI.getCurrentUser();
        if (userData && userData.id) {
          setUser(userData);
          setSessionVerified(true);
        } else {
          setUser(null);
          setSessionVerified(false);
        }
      } catch (error) {
        console.log('Session verification:', error);
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
    
    // переподключает WebSocket после успешного логина
    wsService.reconnectAfterAuth();
  };

  const register = async (email: string, username: string, password: string) => {
    setUser(null);
    setSessionVerified(false);
    
    const userData = await authAPI.register(email, username, password);
    setUser(userData);
    setSessionVerified(true);
    setLoading(false);
    
    wsService.reconnectAfterAuth();
  };

  const logout = async () => {
    try {
      // завершаем все активные сессии перед выходом
      try {
        const { sessionsAPI } = await import('../services/api');
        const sessions = await sessionsAPI.getSessions();
        const activeSessions = sessions.filter((s: any) => s.status === 'active');

        if (activeSessions.length > 0) {
          const endPromises = activeSessions.map(session =>
            sessionsAPI.endSession(session.id).catch(err =>
              console.warn(`Failed to end session ${session.id}:`, err)
            )
          );
          await Promise.allSettled(endPromises);
        }
      } catch (error) {
        console.warn('Failed to auto-end sessions during logout:', error);
      }

      await authAPI.logout();
      wsService.disconnect();
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
