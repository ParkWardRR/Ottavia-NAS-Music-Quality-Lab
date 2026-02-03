// Ottavia - Music Quality Lab
// Main application JavaScript

// Alpine.js store for global state
document.addEventListener('alpine:init', () => {
  Alpine.store('router', {
    path: window.location.pathname
  });

  Alpine.store('theme', {
    current: localStorage.getItem('theme') || 'system',

    init() {
      this.apply();
      window.matchMedia('(prefers-color-scheme: dark)').addEventListener('change', () => {
        if (this.current === 'system') {
          this.apply();
        }
      });
    },

    set(theme) {
      this.current = theme;
      localStorage.setItem('theme', theme);
      this.apply();
    },

    apply() {
      const isDark = this.current === 'dark' ||
        (this.current === 'system' && window.matchMedia('(prefers-color-scheme: dark)').matches);
      document.documentElement.classList.toggle('dark', isDark);
    }
  });

  Alpine.store('notifications', {
    items: [],

    add(message, type = 'info', duration = 5000) {
      const id = Date.now();
      this.items.push({ id, message, type });

      if (duration > 0) {
        setTimeout(() => this.remove(id), duration);
      }

      return id;
    },

    remove(id) {
      this.items = this.items.filter(n => n.id !== id);
    },

    success(message) {
      return this.add(message, 'success');
    },

    error(message) {
      return this.add(message, 'error', 0);
    },

    warning(message) {
      return this.add(message, 'warning');
    }
  });
});

// Main app component
function app() {
  return {
    sidebarOpen: true,
    loading: false,

    init() {
      // Initialize theme
      Alpine.store('theme').init();

      // Setup HTMX event listeners
      this.setupHTMX();

      // Check for system dark mode
      this.checkDarkMode();
    },

    checkDarkMode() {
      const theme = localStorage.getItem('theme') || 'system';
      if (theme === 'system') {
        const isDark = window.matchMedia('(prefers-color-scheme: dark)').matches;
        document.documentElement.classList.toggle('dark', isDark);
      } else {
        document.documentElement.classList.toggle('dark', theme === 'dark');
      }
    },

    setupHTMX() {
      document.body.addEventListener('htmx:beforeRequest', () => {
        this.loading = true;
      });

      document.body.addEventListener('htmx:afterRequest', () => {
        this.loading = false;
      });

      document.body.addEventListener('htmx:responseError', (e) => {
        this.loading = false;
        Alpine.store('notifications').error('An error occurred. Please try again.');
      });

      document.body.addEventListener('htmx:afterSwap', (e) => {
        // Re-initialize any components after swap
        Alpine.store('router').path = window.location.pathname;
      });
    },

    toggleSidebar() {
      this.sidebarOpen = !this.sidebarOpen;
      localStorage.setItem('sidebarOpen', this.sidebarOpen);
    },

    formatBytes(bytes) {
      if (bytes === 0) return '0 B';
      const k = 1024;
      const sizes = ['B', 'KB', 'MB', 'GB', 'TB'];
      const i = Math.floor(Math.log(bytes) / Math.log(k));
      return parseFloat((bytes / Math.pow(k, i)).toFixed(1)) + ' ' + sizes[i];
    },

    formatDuration(seconds) {
      const mins = Math.floor(seconds / 60);
      const secs = Math.floor(seconds % 60);
      return `${mins}:${secs.toString().padStart(2, '0')}`;
    },

    formatDate(dateString) {
      const date = new Date(dateString);
      return date.toLocaleDateString('en-US', {
        month: 'short',
        day: 'numeric',
        hour: '2-digit',
        minute: '2-digit'
      });
    }
  };
}

// API helper functions
const api = {
  baseUrl: '/api',

  async request(endpoint, options = {}) {
    const url = `${this.baseUrl}${endpoint}`;
    const config = {
      headers: {
        'Content-Type': 'application/json',
        ...options.headers
      },
      ...options
    };

    try {
      const response = await fetch(url, config);
      if (!response.ok) {
        throw new Error(`HTTP error! status: ${response.status}`);
      }
      return await response.json();
    } catch (error) {
      console.error('API request failed:', error);
      throw error;
    }
  },

  get(endpoint) {
    return this.request(endpoint, { method: 'GET' });
  },

  post(endpoint, data) {
    return this.request(endpoint, {
      method: 'POST',
      body: JSON.stringify(data)
    });
  },

  put(endpoint, data) {
    return this.request(endpoint, {
      method: 'PUT',
      body: JSON.stringify(data)
    });
  },

  delete(endpoint) {
    return this.request(endpoint, { method: 'DELETE' });
  },

  // Libraries
  getLibraries() {
    return this.get('/libraries');
  },

  getLibrary(id) {
    return this.get(`/libraries/${id}`);
  },

  createLibrary(data) {
    return this.post('/libraries', data);
  },

  scanLibrary(id) {
    return this.post(`/libraries/${id}/scan`);
  },

  // Tracks
  getTracks(params = {}) {
    const query = new URLSearchParams(params).toString();
    return this.get(`/tracks${query ? '?' + query : ''}`);
  },

  getTrack(id) {
    return this.get(`/tracks/${id}`);
  },

  // Settings
  getSettings() {
    return this.get('/settings');
  },

  updateSettings(data) {
    return this.post('/settings', data);
  },

  // Stats
  getDashboardStats() {
    return this.get('/stats');
  },

  // Jobs
  getJobs(status = '') {
    return this.get(`/jobs${status ? '?status=' + status : ''}`);
  }
};

// Utility functions
function debounce(func, wait) {
  let timeout;
  return function executedFunction(...args) {
    const later = () => {
      clearTimeout(timeout);
      func(...args);
    };
    clearTimeout(timeout);
    timeout = setTimeout(later, wait);
  };
}

function throttle(func, limit) {
  let inThrottle;
  return function executedFunction(...args) {
    if (!inThrottle) {
      func.apply(this, args);
      inThrottle = true;
      setTimeout(() => inThrottle = false, limit);
    }
  };
}

// Copy to clipboard
async function copyToClipboard(text) {
  try {
    await navigator.clipboard.writeText(text);
    Alpine.store('notifications').success('Copied to clipboard');
    return true;
  } catch (err) {
    Alpine.store('notifications').error('Failed to copy');
    return false;
  }
}

// Download file
function downloadFile(url, filename) {
  const link = document.createElement('a');
  link.href = url;
  link.download = filename;
  document.body.appendChild(link);
  link.click();
  document.body.removeChild(link);
}

// Export functions for global use
window.app = app;
window.api = api;
window.copyToClipboard = copyToClipboard;
window.downloadFile = downloadFile;
