import React, { useState, useEffect } from 'react';
import { useNavigate } from 'react-router-dom';
import { useAuth } from '../contexts/AuthContext';
import { DecorativeElements } from './DecorativeElements';
import './LoginForm.css';

export const LoginForm: React.FC = () => {
  const { login, register, isAuthenticated, loading: authLoading } = useAuth();
  const navigate = useNavigate();
  const [isLogin, setIsLogin] = useState(true);
  const [email, setEmail] = useState('');
  const [username, setUsername] = useState('');
  const [password, setPassword] = useState('');
  const [error, setError] = useState<string | null>(null);
  const [loading, setLoading] = useState(false);

  useEffect(() => {
    if (!authLoading && isAuthenticated) {
      navigate('/');
    }
  }, [isAuthenticated, authLoading, navigate]);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setError(null);
    setLoading(true);

    try {
      if (isLogin) {
        await login(email, password);
      } else {
        if (!username.trim()) {
          setError('Требуется имя пользователя');
          setLoading(false);
          return;
        }
        await register(email, username, password);
      }
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Ошибка аутентификации');
      setLoading(false);
    }
  };

  return (
    <div className="login-container">
      <DecorativeElements />
      <div className="login-card">
        <h2>{isLogin ? 'Вход' : 'Регистрация'}</h2>
        <form onSubmit={handleSubmit}>
          {!isLogin && (
            <div className="form-group">
              <label htmlFor="username">Username</label>
              <input
                id="username"
                type="text"
                value={username}
                onChange={(e) => setUsername(e.target.value)}
                required
                placeholder="Введите имя пользователя"
                minLength={3}
                maxLength={30}
              />
            </div>
          )}
          <div className="form-group">
            <label htmlFor="email">Адрес электронной почты</label>
            <input
              id="email"
              type="email"
              value={email}
              onChange={(e) => setEmail(e.target.value)}
              required
              placeholder="Введите почту"
            />
          </div>
          <div className="form-group">
            <label htmlFor="password">Пароль</label>
            <input
              id="password"
              type="password"
              value={password}
              onChange={(e) => setPassword(e.target.value)}
              required
              placeholder="Введите пароль"
              minLength={8}
            />
            {!isLogin && (
              <small>Пароль должен состоять из 8 и более символов, включая хотя бы одну букву и одну цифру</small>
            )}
          </div>
          {error && <div className="error-message">{error}</div>}
          <button type="submit" className="btn btn-primary btn-block" disabled={loading}>
            {loading ? 'Обработка...' : isLogin ? 'Вход' : 'Регистрация'}
          </button>
        </form>
        <div className="form-footer">
          <button
            type="button"
            onClick={() => {
              setIsLogin(!isLogin);
              setError(null);
              setEmail('');
              setPassword('');
              setUsername('');
            }}
            className="btn-link"
          >
            {isLogin ? "Нет аккаунта? Регистрация" : 'Уже есть аккаунт? Вход'}
          </button>
        </div>
      </div>
    </div>
  );
};



