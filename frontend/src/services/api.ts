import axios, { AxiosError } from 'axios';
import type { User, Session, Event, CreateSessionRequest, CreateEventRequest } from '../types';

const api = axios.create({
  baseURL: '/api',
  withCredentials: true,
  headers: {
    'Content-Type': 'application/json',
  },
});

const getErrorMessage = (error: unknown): string => {
  if (axios.isAxiosError(error)) {
    const axiosError = error as AxiosError;
    if (axiosError.response) {
      const data = axiosError.response.data;
      if (typeof data === 'string') {
        return data;
      }
      if (data && typeof data === 'object') {
        const obj = data as any;
        if (obj.error) {
          return obj.error;
        }
        if (obj.message) {
          return obj.message;
        }
      }
      return axiosError.response.statusText || 'Запрос не выполнен';
    }
    if (axiosError.request) {
      return 'Ошибка сети: Не удалось подключиться к серверу';
    }
  }
  if (error instanceof Error) {
    return error.message;
  }
  return 'Произошла неожиданная ошибка';
};

export const authAPI = {
  login: async (email: string, password: string): Promise<User> => {
    try {
      const response = await api.post<User>('/auth/login', { email, password });
      return response.data;
    } catch (error) {
      throw new Error(getErrorMessage(error));
    }
  },

  register: async (email: string, username: string, password: string): Promise<User> => {
    try {
      const response = await api.post<User>('/auth/register', { email, username, password });
      return response.data;
    } catch (error) {
      throw new Error(getErrorMessage(error));
    }
  },

  getCurrentUser: async (): Promise<User> => {
    try {
      const response = await api.get<User>('/auth/me');
      return response.data;
    } catch (error) {
      throw new Error(getErrorMessage(error));
    }
  },

  logout: async (): Promise<void> => {
    try {
      await api.post('/auth/logout');
    } catch (error) {
      console.error('Logout API error:', error);
    }
  },
};

export const sessionsAPI = {
  getSessions: async (): Promise<Session[]> => {
    try {
      const response = await api.get<Session[]>('/sessions');
      return Array.isArray(response.data) ? response.data : [];
    } catch (error) {
      throw new Error(getErrorMessage(error));
    }
  },

  createSession: async (notes?: string): Promise<Session> => {
    try {
      const response = await api.post<Session>('/sessions', { notes } as CreateSessionRequest);
      return response.data;
    } catch (error) {
      throw new Error(getErrorMessage(error));
    }
  },

  endSession: async (sessionId: number): Promise<void> => {
    try {
      await api.post(`/sessions/end?id=${sessionId}`);
    } catch (error) {
      throw new Error(getErrorMessage(error));
    }
  },

  deleteSession: async (sessionId: number): Promise<void> => {
    try {
      await api.post(`/sessions/delete?id=${sessionId}`);
    } catch (error) {
      throw new Error(getErrorMessage(error));
    }
  },
};

export const eventsAPI = {
  getEvents: async (sessionId: number): Promise<Event[]> => {
    try {
      const response = await api.get<Event[]>(`/events?session_id=${sessionId}`);
      return Array.isArray(response.data) ? response.data : [];
    } catch (error) {
      throw new Error(getErrorMessage(error));
    }
  },

  saveEvent: async (event: CreateEventRequest): Promise<Event> => {
    try {
      const response = await api.post<Event>('/events', event);
      return response.data;
    } catch (error) {
      throw new Error(getErrorMessage(error));
    }
  },
};

export const healthAPI = {
  check: async (): Promise<{ status: string; grpc_ok: boolean; http_ok: boolean }> => {
    try {
      const response = await api.get('/health');
      return response.data;
    } catch (error) {
      throw new Error(getErrorMessage(error));
    }
  },
};



