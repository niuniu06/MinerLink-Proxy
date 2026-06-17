<template>
  <div class="svg-coin-icon" :style="{ backgroundColor: bgCol }">
    <svg v-if="c === 'BTC'" viewBox="0 0 32 32" fill="none" xmlns="http://www.w3.org/2000/svg">
      <circle cx="16" cy="16" r="16" fill="#F7931A"/>
      <path d="M22.5 13.9C22.7 12.5 21.6 11.8 19.5 11.1L20.8 5.8L18.4 5.2L17.2 10.3C16.6 10.2 16 10 15.4 9.9L16.6 4.7L14.2 4.1L12.9 9.3C12.4 9.2 11.9 9.1 11.4 8.9L11.4 8.9L8.3 8.1L7.7 10.6C7.7 10.6 9.4 11 9.3 11.1C10.2 11.3 10.4 11.9 10.3 12.3L9 17.5C9.1 17.6 9.3 17.6 9.6 17.7C9.3 17.6 9 17.6 8.7 17.5L7.4 22.8C7.5 23.2 6.9 23.6 6.1 23.4C6.2 23.4 4.5 23 4.5 23L3.3 25.8L6.3 26.5C6.9 26.7 7.5 26.8 8 26.9L6.7 32L9.1 32.6L10.4 27.4C11 27.6 11.6 27.7 12.1 27.9L10.8 33L13.2 33.6L14.5 28.3C18.6 29 21.6 28.3 22.5 24.6C23.2 21.6 22.1 20 20 18.9C21.5 18.6 22.5 17.4 22.5 13.9ZM18.7 23.6C18 26.4 13.5 25.3 11.6 24.8L12.9 19.5C14.8 20 19.5 20.7 18.7 23.6ZM19.5 15.6C18.9 17.9 15.1 16.9 13.5 16.5L14.6 11.8C16.2 12.2 20.1 12.9 19.5 15.6Z" fill="white"/>
    </svg>
    <svg v-else-if="c === 'ETH' || c === 'ETC' || c === 'ETHW'" viewBox="0 0 32 32" fill="none" xmlns="http://www.w3.org/2000/svg">
      <circle cx="16" cy="16" r="16" :fill="c === 'ETC' ? '#328332' : '#627EEA'"/>
      <path d="M15.8 4L7 18.5L15.8 23.7L24.6 18.5L15.8 4Z" fill="white" fill-opacity="0.6"/>
      <path d="M15.8 4L7 18.5L15.8 14.5L15.8 4Z" fill="white"/>
      <path d="M15.8 25L7 19.8L15.8 30L24.6 19.8L15.8 25Z" fill="white" fill-opacity="0.6"/>
      <path d="M15.8 30L15.8 25L7 19.8L15.8 30Z" fill="white"/>
      <path d="M15.8 23.7L24.6 18.5L15.8 14.5L15.8 23.7Z" fill="white" fill-opacity="0.2"/>
      <path d="M7 18.5L15.8 14.5L15.8 23.7L7 18.5Z" fill="white" fill-opacity="0.6"/>
    </svg>
    <svg v-else-if="c === 'LTC'" viewBox="0 0 32 32" fill="none" xmlns="http://www.w3.org/2000/svg">
      <circle cx="16" cy="16" r="16" fill="#345D9D"/>
      <path d="M10.1 23L13 14H10.4L11.1 11.6H13.7L15.4 6H19.5L17.7 11.6H21.4L20.7 14H17L14.7 20.6H23.5L22.8 23H10.1Z" fill="white"/>
    </svg>
    <svg v-else-if="c === 'DOGE'" viewBox="0 0 32 32" fill="none" xmlns="http://www.w3.org/2000/svg">
      <circle cx="16" cy="16" r="16" fill="#C2A633"/>
      <path d="M21.1 11.4C19.7 10 17 9.8 13.9 9.8H9.3V22.2H14C17.4 22.2 19.9 21.9 21.5 20.3C23 18.8 23.4 16.4 23.4 15.6C23.4 14.8 23 13.3 21.1 11.4ZM18.7 18.3C17.7 19.3 16 19.4 14.2 19.4H12.9V12.6H14.1C15.9 12.6 17.5 12.6 18.6 13.7C19.5 14.5 19.6 15.5 19.6 15.9C19.6 16.3 19.5 17.5 18.7 18.3Z" fill="white"/>
      <path d="M10.5 14.7H15.6V17.1H10.5V14.7Z" fill="white"/>
    </svg>
    <svg v-else-if="c === 'KAS'" viewBox="0 0 32 32" fill="none" xmlns="http://www.w3.org/2000/svg">
      <circle cx="16" cy="16" r="16" fill="#70C7BA"/>
      <path d="M10 23L16 13L22 23H25L16 8L7 23H10Z" fill="white"/>
      <path d="M13 23L16 18L19 23H22L16 13L10 23H13Z" fill="white"/>
    </svg>
    <svg v-else-if="c === 'PRL'" viewBox="0 0 32 32" fill="none" xmlns="http://www.w3.org/2000/svg">
      <defs>
        <linearGradient id="prlGrad" x1="0%" y1="0%" x2="100%" y2="100%">
          <stop offset="0%" stop-color="#FF3838" />
          <stop offset="100%" stop-color="#FF930F" />
        </linearGradient>
      </defs>
      <path fill-rule="evenodd" clip-rule="evenodd" d="M2.5 2.5C12 9 20 9 29.5 2.5C23 12 23 20 29.5 29.5C20 23 12 23 2.5 29.5C9 20 9 12 2.5 2.5ZM16 22.5C19.5899 22.5 22.5 19.5899 22.5 16C22.5 12.4101 19.5899 9.5 16 9.5C12.4101 9.5 9.5 12.4101 9.5 16C9.5 19.5899 12.4101 22.5 16 22.5Z" fill="url(#prlGrad)"/>
    </svg>
    <div v-else class="generic-coin-icon">
      {{ c.substring(0, 3) }}
    </div>
  </div>
</template>

<script setup>
import { computed } from 'vue'

const props = defineProps({
  coin: {
    type: String,
    required: true
  }
})

const c = computed(() => {
  if (!props.coin) return 'UNK'
  return props.coin.toUpperCase().trim()
})

const bgCol = computed(() => {
  if (['BTC', 'ETH', 'ETC', 'ETHW', 'LTC', 'DOGE', 'KAS', 'PRL'].includes(c.value)) {
    return 'transparent'
  }
  return 'rgba(88, 166, 255, 0.1)'
})
</script>

<style scoped>
.svg-coin-icon {
  width: 100%;
  height: 100%;
  display: flex;
  align-items: center;
  justify-content: center;
  border-radius: 50%;
  overflow: hidden;
}
.svg-coin-icon svg {
  width: 100%;
  height: 100%;
}
.generic-coin-icon {
  font-size: 14px;
  font-weight: 800;
  color: #fff;
  background: linear-gradient(135deg, var(--accent-blue), #9b59b6);
  -webkit-background-clip: text;
  -webkit-text-fill-color: transparent;
  display: flex;
  align-items: center;
  justify-content: center;
  width: 100%;
  height: 100%;
}
</style>
