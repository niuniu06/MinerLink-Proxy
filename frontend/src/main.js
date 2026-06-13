import { createApp } from 'vue'
import './style.css'
import App from './App.vue'

const originalFetch = window.fetch;
window.fetch = async (resource, config) => {
  if (!config) {
    config = {};
  }
  if (!config.headers) {
    config.headers = {};
  }
  const token = localStorage.getItem('mlp_token') || sessionStorage.getItem('mlp_token');
  if (token) {
    config.headers['Authorization'] = `Bearer ${token}`;
  }
  const response = await originalFetch(resource, config);
  if (response.status === 401 && resource !== '/api/login') {
    localStorage.removeItem('mlp_token');
    sessionStorage.removeItem('mlp_token');
    window.location.reload();
  }
  return response;
};

createApp(App).mount('#app')
