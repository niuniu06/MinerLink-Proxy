import { createApp } from 'vue'
import './style.css'
import App from './App.vue'

const originalFetch = window.fetch;
window.fetch = async (resource, config) => {
  if (!config) {
    config = {};
  }
  
  const token = localStorage.getItem('mlp_token') || sessionStorage.getItem('mlp_token');
  
  if (token) {
    if (!config.headers) {
      config.headers = {};
    }
    // Handle both plain object and Headers instance safely
    if (config.headers instanceof Headers) {
      config.headers.set('Authorization', `Bearer ${token}`);
    } else {
      config.headers = { ...config.headers, 'Authorization': `Bearer ${token}` };
    }
  }

  try {
    const response = await originalFetch.call(window, resource, config);
    if (response.status === 401 && resource !== '/api/login') {
      localStorage.removeItem('mlp_token');
      sessionStorage.removeItem('mlp_token');
      window.location.reload();
      // Return a promise that never resolves to stop execution of the caller
      return new Promise(() => {});
    }
    return response;
  } catch (e) {
    throw e;
  }
};

createApp(App).mount('#app')
