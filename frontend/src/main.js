import { mount } from 'svelte'
import './style.css' // Garanta que este arquivo existe ou remova esta linha
import App from './App.svelte'

// No Svelte 5, usamos 'mount' em vez de 'new App'
const app = mount(App, {
  target: document.getElementById('app'),
})

export default app