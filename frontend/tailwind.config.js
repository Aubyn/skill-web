/** @type {import('tailwindcss').Config} */
export default {
  content: [
    "./index.html",
    "./src/**/*.{js,ts,jsx,tsx}",
  ],
  theme: {
    extend: {
      colors: {
        bg: {
          base: '#f7f5f0',
          muted: '#eeebe4',
          card: '#ffffff',
        },
        fg: {
          base: '#1f1d1a',
          muted: '#6b6560',
          subtle: '#9e9690',
        },
        accent: {
          DEFAULT: '#d8604a',
          light: '#f09a86',
          dark: '#b54735',
          bg: '#fef0eb',
        },
        border: {
          DEFAULT: '#e5ddd4',
          hover: '#d4cabe',
        },
        sidebar: {
          bg: '#eeebe4',
          hover: '#e3ddd4',
          active: '#d8cfc4',
          text: '#5a5550',
          'text-active': '#1f1d1a',
        },
      },
      fontFamily: {
        display: ['"Noto Sans SC"', '"PingFang SC"', '"Microsoft YaHei"', 'sans-serif'],
        body: ['"Noto Sans SC"', '"PingFang SC"', '"Microsoft YaHei"', 'sans-serif'],
        mono: ['"JetBrains Mono"', '"Fira Code"', 'monospace'],
      },
      borderRadius: {
        sm: '4px',
        DEFAULT: '7px',
        md: '10px',
        lg: '14px',
        xl: '18px',
      },
      boxShadow: {
        card: '0 1px 2px rgba(31,29,26,0.05), 0 0 0 1px rgba(31,29,26,0.03)',
        'card-hover': '0 8px 24px rgba(31,29,26,0.10), 0 2px 6px rgba(31,29,26,0.06)',
        sidebar: '1px 0 0 rgba(31,29,26,0.05)',
        dialog: '0 12px 40px rgba(31,29,26,0.14), 0 4px 12px rgba(31,29,26,0.06)',
        'btn': '0 1px 2px rgba(31,29,26,0.08)',
      },
      spacing: {
        'sidebar': '220px',
      },
    },
  },
  plugins: [],
}
