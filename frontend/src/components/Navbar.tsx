import React from 'react';
import { useNavigate } from 'react-router-dom';
import { useAuth } from '../contexts/AuthContext';
import './Navbar.css';

export const Navbar: React.FC = () => {
  const { user, logout, isAuthenticated } = useAuth();
  const navigate = useNavigate();

  const handleLogout = async () => {
    try {
      await logout();
      navigate('/login');
    } catch (error) {
      console.error('Logout failed:', error);
    }
  };

  if (!isAuthenticated) {
    return null;
  }

  return (
    <nav className="navbar">
      <div className="navbar-brand">
        <span>🌠</span>
        <span>Детектор сонливости</span>
      </div>
      <div className="navbar-right">
        <span className="user-name">{user?.username || user?.email}</span>
        <button onClick={handleLogout} className="btn-logout">
          Выйти
        </button>
      </div>
    </nav>
  );
};
