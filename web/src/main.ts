import { mount } from 'svelte'
import './app.css'
import App from './App.svelte'

// Register Progressive Web App Service Worker
if ('serviceWorker' in navigator) {
  window.addEventListener('load', () => {
    navigator.serviceWorker
      .register('./sw.js', { scope: './' })
      .then((reg) => {
        reg.onupdatefound = () => {
          const installingWorker = reg.installing;
          if (installingWorker) {
            installingWorker.onstatechange = () => {
              if (installingWorker.state === 'installed' && navigator.serviceWorker.controller) {
                console.log('[gofyr PWA] New update available.');
              }
            };
          }
        };
      })
      .catch((err) => {
        console.warn('[gofyr PWA] Service worker registration skipped/failed:', err);
      });
  });
}

const app = mount(App, {
  target: document.getElementById('app')!,
})

export default app
