import {defineConfig} from 'vite'
import {svelte} from '@sveltejs/vite-plugin-svelte'

// https://vitejs.dev/config/
export default defineConfig({
  plugins: [svelte({
    onwarn: (warning, handler) => {
      // Ignora warnings de acessibilidade (é um app desktop, não web)
      if (warning.code && warning.code.startsWith('a11y')) return;
      handler(warning);
    }
  })]
})
