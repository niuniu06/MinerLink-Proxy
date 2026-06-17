<template>
  <div class="login-container">
    <div class="login-card">
      <div class="logo-box">
        <img src="/favicon.svg" alt="MinerLink-Proxy Logo" class="logo-img" />
        <h2>MinerLink-Proxy</h2>
      </div>
      <form @submit.prevent="handleLogin" class="login-form">
        <div class="input-group">
          <input type="text" v-model="account" placeholder="请输入账号" required />
        </div>
        <div class="input-group">
          <input type="password" v-model="password" placeholder="请输入密码" required />
        </div>
        <div class="remember-group">
          <input type="checkbox" id="remember" v-model="remember" />
          <label for="remember">记住登录状态</label>
        </div>
        <div class="error-msg" v-if="error">{{ error }}</div>
        <button type="submit" class="btn-login" :disabled="loading">
          {{ loading ? '登录中...' : '立即登录' }}
        </button>
      </form>
    </div>
  </div>
</template>

<script setup>
import { ref } from 'vue'

const emit = defineEmits(['success'])

const account = ref('')
const password = ref('')
const remember = ref(false)
const error = ref('')
const loading = ref(false)

const handleLogin = async () => {
  error.value = ''
  loading.value = true
  try {
    const res = await fetch('/api/login', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ account: account.value, password: password.value })
    })
    const data = await res.json()
    if (res.ok && data.success) {
      if (remember.value) {
        localStorage.setItem('mlp_token', data.token)
      } else {
        sessionStorage.setItem('mlp_token', data.token)
      }
      emit('success')
    } else {
      error.value = data.error || '账号或密码错误'
    }
  } catch (e) {
    error.value = '网络请求失败，请检查后端状态'
  } finally {
    loading.value = false
  }
}
</script>

<style scoped>
.login-container {
  min-height: 100vh;
  display: flex;
  align-items: center;
  justify-content: center;
  background-color: var(--bg-color, #0d1117);
  font-family: 'Inter', system-ui, sans-serif;
}

.login-card {
  width: 100%;
  max-width: 360px;
  background: transparent;
  padding: 40px 20px;
  display: flex;
  flex-direction: column;
  align-items: center;
}

.logo-box {
  text-align: center;
  margin-bottom: 40px;
}

.logo-img {
  width: 80px;
  height: 80px;
  border-radius: 50%;
  object-fit: cover;
  margin-bottom: 16px;
  box-shadow: 0 4px 20px rgba(0, 0, 0, 0.5);
  background-color: #fff;
  padding: 4px;
}

.logo-box h2 {
  color: #ffffff;
  font-size: 24px;
  font-weight: 700;
  margin: 0;
  letter-spacing: 0.5px;
  font-style: italic;
}

.login-form {
  width: 100%;
  display: flex;
  flex-direction: column;
  gap: 20px;
}

.input-group input {
  width: 100%;
  background: #ffffff;
  border: none;
  border-radius: 24px;
  padding: 14px 20px;
  font-size: 14px;
  color: #333;
  outline: none;
  box-shadow: 0 4px 6px rgba(0,0,0,0.1);
  transition: box-shadow 0.2s;
  box-sizing: border-box;
}

.input-group input:focus {
  box-shadow: 0 0 0 2px rgba(88, 166, 255, 0.5);
}

.remember-group {
  display: flex;
  align-items: center;
  gap: 8px;
  color: #fff;
  font-size: 13px;
  margin-top: -10px;
  padding-left: 10px;
}

.error-msg {
  color: #ff6b6b;
  font-size: 12px;
  text-align: center;
}

.btn-login {
  width: 100%;
  background: linear-gradient(135deg, #6c79ab, #525f8f);
  color: white;
  border: none;
  border-radius: 24px;
  padding: 14px;
  font-size: 15px;
  font-weight: 600;
  cursor: pointer;
  margin-top: 10px;
  box-shadow: 0 4px 15px rgba(0,0,0,0.2);
  transition: transform 0.1s, box-shadow 0.2s;
}

.btn-login:hover {
  transform: translateY(-2px);
  box-shadow: 0 6px 20px rgba(0,0,0,0.3);
}

.btn-login:active {
  transform: translateY(0);
}
</style>
