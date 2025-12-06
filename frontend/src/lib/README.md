# GoAnime Frontend - Estrutura de Código

## 📁 Estrutura de Pastas

```
src/
├── lib/                          # Biblioteca principal
│   ├── components/               # Componentes Svelte
│   │   ├── layout/               # Componentes de layout
│   │   │   ├── Header.svelte     # Barra de navegação principal
│   │   │   └── NavTabs.svelte    # Abas de navegação (Anime, Manga, etc)
│   │   ├── ui/                   # Componentes de UI genéricos
│   │   │   ├── SearchBar.svelte  # Barra de busca
│   │   │   ├── SplashScreen.svelte # Tela de loading inicial
│   │   │   └── LoadingSpinner.svelte # Indicador de carregamento
│   │   ├── anime/                # Componentes específicos de anime
│   │   │   ├── AnimeCard.svelte  # Card de anime individual
│   │   │   ├── AnimeGrid.svelte  # Grid responsivo de cards
│   │   │   ├── GenreFilter.svelte # Filtro por gêneros
│   │   │   └── FeaturedHero.svelte # Seção hero com destaque
│   │   ├── views/                # Views/páginas principais
│   │   │   └── LoginScreen.svelte # Tela de login
│   │   ├── modals/               # Modais
│   │   │   └── DiscordLinkModal.svelte # Modal de vinculação Discord
│   │   └── index.js              # Re-exporta todos os componentes
│   │
│   ├── stores/                   # Estados globais (Svelte 5 runes)
│   │   └── index.svelte.js       # Stores: user, settings, discord, ui, anime, player
│   │
│   ├── services/                 # Serviços de API
│   │   ├── anime.js              # Chamadas ao backend: busca, episódios, etc
│   │   ├── user.js               # Gerenciamento de usuário e favoritos
│   │   ├── discord.js            # Integração Discord
│   │   └── index.js              # Re-exporta todos os serviços
│   │
│   ├── utils/                    # Funções utilitárias
│   │   └── helpers.js            # formatTimeAgo, debounce, throttle, etc
│   │
│   ├── constants/                # Constantes e configurações
│   │   └── genres.js             # Lista de gêneros, avatares, settings padrão
│   │
│   └── index.js                  # Exporta tudo da lib
│
├── App.svelte                    # Componente raiz
├── main.js                       # Entry point
└── style.css                     # Estilos globais
```

## 🎯 Padrões de Código

### Componentes
- Cada componente tem uma única responsabilidade
- Props tipadas com JSDoc
- Eventos via callbacks (onXxx)
- Estilos encapsulados (scoped)

### Stores (Svelte 5)
- Usando `$state` e `$derived` para reatividade
- Getters para leitura, métodos para escrita
- Separados por domínio (user, anime, player, etc)

### Serviços
- Encapsulam chamadas ao backend Go (Wails)
- Tratamento de erros consistente
- Funções async/await

### Utilitários
- Funções puras e reutilizáveis
- Bem documentadas com JSDoc

## 🚀 Como Usar

### Importando componentes:
```js
import { Header, NavTabs, AnimeCard } from './lib/components/index.js';
```

### Usando stores:
```js
import { userStore, animeStore, playerStore } from './lib/stores/index.svelte.js';

// Leitura (reativa)
const user = userStore.user;

// Escrita
userStore.setUser({ username: 'João' });
```

### Usando serviços:
```js
import { animeService, userService } from './lib/services/index.js';

const animes = await animeService.searchAnimes('naruto');
const favorites = await userService.getFavorites();
```

## 📦 Adicionando Novos Componentes

1. Crie o arquivo `.svelte` na pasta apropriada
2. Exporte no `index.js` da pasta
3. Use tipagem JSDoc para props
4. Mantenha estilos encapsulados

## 🔧 Manutenção

- **Stores**: Adicione novos estados em `stores/index.svelte.js`
- **Serviços**: Adicione novas chamadas API em `services/`
- **Constantes**: Adicione configurações em `constants/`
