import React, { useState, useEffect } from 'react';
import { sessionsAPI } from '../services/api';
import type { Session } from '../types';
import './SessionManager.css';

interface SessionManagerProps {
  onSessionSelect?: (sessionId: number | null) => void;
  currentSessionId?: number | null;
  onSessionCreated?: (sessionId: number) => void;
  onViewSession?: (sessionId: number) => void;
  onEndSession?: (sessionId: number) => void;
}

export const SessionManager: React.FC<SessionManagerProps> = ({
                                                                onSessionSelect,
                                                                currentSessionId,
                                                                onSessionCreated,
                                                                onViewSession,
                                                                onEndSession,
                                                              }) => {
  const [sessions, setSessions] = useState<Session[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [showCreateForm, setShowCreateForm] = useState(false);
  const [notes, setNotes] = useState('');

  // Пагинация
  const [currentPage, setCurrentPage] = useState(1);
  const sessionsPerPage = 5;

  const loadSessions = async () => {
    setLoading(true);
    setError(null);
    try {
      const data = await sessionsAPI.getSessions();
      setSessions(Array.isArray(data) ? data : []);
      setCurrentPage(1); // Сброс на первую страницу при обновлении
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Не удалось загрузить сеансы');
      setSessions([]);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    loadSessions();
  }, []);

  const formatDateTime = (dateString: string) => {
    try {
      return new Date(dateString).toLocaleString('ru-RU', {
        day: '2-digit',
        month: '2-digit',
        year: 'numeric',
        hour: '2-digit',
        minute: '2-digit',
        second: '2-digit',
        timeZone: 'Europe/Moscow',
      });
    } catch {
      return dateString;
    }
  };

  const handleCreateSession = async (e: React.FormEvent) => {
    e.preventDefault();
    setError(null);

    const sessionNotes = notes.trim() || `Поездка: ${formatDateTime(new Date())}`;

    try {
      const newSession = await sessionsAPI.createSession(sessionNotes);
      setSessions([newSession, ...sessions]);
      setNotes('');
      setShowCreateForm(false);
      onSessionSelect?.(newSession.id);
      onSessionCreated?.(newSession.id);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Не удалось создать сеанс');
    }
  };

  const handleEndSession = async (sessionId: number) => {
    if (!confirm('Вы уверены, что хотите завершить эту сессию и выключить камеру?')) {
      return;
    }

    setError(null);
    try {
      if (currentSessionId === sessionId && onEndSession) {
        await onEndSession(sessionId);
        await loadSessions();
      } else {
        await sessionsAPI.endSession(sessionId);
        await loadSessions();
      }
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Не удалось завершить сеанс');
    }
  };

  const handleDeleteSession = async (sessionId: number) => {
    if (!confirm('Вы уверены, что хотите удалить этот сеанс? Это действие нельзя отменить.')) {
      return;
    }

    setError(null);
    try {
      await sessionsAPI.deleteSession(sessionId);
      await loadSessions();
      setSessions(sessions.filter(session => session.id !== sessionId));
      if (currentSessionId === sessionId && onSessionSelect) {
        onSessionSelect(null);
      }
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Не удалось удалить сеанс');
      await loadSessions();
    }
  };

  const formatDate = (dateString: string) => {
    try {
      return new Date(dateString).toLocaleString('ru-RU', {
        day: '2-digit',
        month: '2-digit',
        year: 'numeric',
        hour: '2-digit',
        minute: '2-digit',
        second: '2-digit',
        timeZone: 'Europe/Moscow',
      });
    } catch {
      return dateString;
    }
  };

  const getDuration = (startTime: string, endTime?: string) => {
    const start = new Date(startTime);
    const end = endTime ? new Date(endTime) : new Date();
    const diff = Math.floor((end.getTime() - start.getTime()) / 1000);
    const hours = Math.floor(diff / 3600);
    const minutes = Math.floor((diff % 3600) / 60);
    const seconds = diff % 60;
    return `${hours}ч ${minutes}м ${seconds}с`;
  };

  // Пагинация
  const totalPages = Math.ceil(sessions.length / sessionsPerPage);
  const startIndex = (currentPage - 1) * sessionsPerPage;
  const currentSessions = sessions.slice(startIndex, startIndex + sessionsPerPage);

  const handleNextPage = () => {
    if (currentPage < totalPages) {
      setCurrentPage(currentPage + 1);
    }
  };

  const handlePrevPage = () => {
    if (currentPage > 1) {
      setCurrentPage(currentPage - 1);
    }
  };

  const handleFirstPage = () => {
    setCurrentPage(1);
  };

  return (
      <div className="session-manager">
        <div className="session-header">
          <h2>Поездки</h2>
          <button
              onClick={() => setShowCreateForm(!showCreateForm)}
              className="btn btn-primary"
          >
            {showCreateForm ? 'Назад' : 'Добавить'}
          </button>
        </div>

        {showCreateForm && (
            <form onSubmit={handleCreateSession} className="session-form">
              <div className="form-group">
                <label htmlFor="notes">Название (необязательно):</label>
                <textarea
                    id="notes"
                    value={notes}
                    onChange={(e) => setNotes(e.target.value)}
                    placeholder="Добавьте информацию о вашей поездке..."
                    rows={3}
                />
              </div>
              <button type="submit" className="btn btn-primary">
                Создать сеанс
              </button>
            </form>
        )}

        {error && (
            <div className="error-message">
              {error}
              <button onClick={() => setError(null)} className="btn-close">×</button>
            </div>
        )}

        {loading ? (
            <div className="loading">Загрузка сеансов...</div>
        ) : !sessions || sessions.length === 0 ? (
            <div className="empty-state">
              <p>Нажмите "Добавить", чтобы начать первую поездку.</p>
            </div>
        ) : (
            <>
              <div className="session-list">
                {currentSessions.map((session) => (
                    <div
                        key={session.id}
                        className={`session-item ${currentSessionId === session.id ? 'active' : ''} ${
                            session.status === 'active' ? 'status-active' : 'status-completed'
                        }`}
                    >
                      <div className="session-info">
                        <div className="session-id">
                          {session.notes || `Сеанс`}
                        </div>
                        <div className="session-time">
                          Начало: {formatDate(session.start_time)}
                        </div>
                        {session.end_time && (
                            <div className="session-time">
                              Конец: {formatDate(session.end_time)}
                            </div>
                        )}
                        <div className="session-duration">
                          Время поездки: {getDuration(session.start_time, session.end_time)}
                        </div>
                        <div className="session-status">
                    <span className={`status-badge ${session.status}`}>
                      {session.status === 'completed' || session.status === 'завершен' ? 'завершен' : session.status}
                    </span>
                        </div>
                      </div>
                      <div className="session-actions">
                        {session.status === 'active' && (
                            <>
                              {currentSessionId !== session.id && (
                                  <button
                                      onClick={() => onSessionSelect?.(session.id)}
                                      className="btn btn-secondary btn-sm"
                                  >
                                    Выбрать
                                  </button>
                              )}
                              <button
                                  onClick={() => handleEndSession(session.id)}
                                  className="btn btn-danger btn-sm"
                              >
                                Конец
                              </button>
                            </>
                        )}
                        {(session.status === 'completed' || session.status === 'завершен') && (
                            <>
                              <button
                                  onClick={() => onViewSession?.(session.id)}
                                  className="btn btn-secondary btn-sm"
                              >
                                Смотреть
                              </button>
                              <button
                                  onClick={() => handleDeleteSession(session.id)}
                                  className="btn btn-danger btn-sm"
                                  title="Удалить сеанс"
                              >
                                🗑️
                              </button>
                            </>
                        )}
                      </div>
                    </div>
                ))}
              </div>

              {/* Пагинация */}
              <div className="pagination">
                <button onClick={handleFirstPage} disabled={currentPage === 1}>
                  В начало
                </button>
                <button onClick={handlePrevPage} disabled={currentPage === 1}>
                  Предыдущая
                </button>
                <span>
              Страница {currentPage} из {totalPages}
            </span>
                <button onClick={handleNextPage} disabled={currentPage === totalPages}>
                  Следующая
                </button>
              </div>
            </>
        )}
      </div>
  );
};