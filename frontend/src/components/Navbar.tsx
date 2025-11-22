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
      <div className="navbar-content">
        <div className="navbar-brand">
          <h2>🌠 Детектор сонливости</h2>
        </div>
        <div className="navbar-user">
          <span className="user-info">
            {user?.username || user?.email}
          </span>
          <button onClick={handleLogout} className="btn btn-secondary btn-sm">
            Выйти
          </button>
        </div>
      </div>
    </nav>
  );
};




















