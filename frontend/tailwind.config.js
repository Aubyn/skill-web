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
          base: '#faf8f5',
          muted: '#f0ece4',
          card: '#ffffff',
        },
        fg: {
          base: '#2c2420',
          muted: '#7a7068',
          subtle: '#a89e94',
        },
        accent: {
          DEFAULT: '#c45d42',
          light: '#e88b6e',
          dark: '#a34a32',
          bg: '#fdf0ea',
        },
        border: {
          DEFAULT: '#e8e0d6',
          hover: '#d4cabc',
        },
        sidebar: {
          bg: '#f0ece4',
          hover: '#e6dfd4',
          active: '#d4cabc',
          text: '#5a5048',
          'text-active': '#2c2420',
        },
      },
      fontFamily: {
        display: ['"Noto Serif SC"', '"Georgia"', 'serif'],
        body: ['"Noto Sans SC"', '"PingFang SC"', '"Microsoft YaHei"', 'sans-serif'],
        mono: ['"JetBrains Mono"', '"Fira Code"', 'monospace'],
      },
      borderRadius: {
        sm: '4px',
        DEFAULT: '6px',
        md: '8px',
        lg: '12px',
      },
      boxShadow: {
        card: '0 1px 3px rgba(44,36,32,0.06), 0 1px 2px rgba(44,36,32,0.04)',
        'card-hover': '0 4px 12px rgba(44,36,32,0.08), 0 2px 4px rgba(44,36,32,0.04)',
        sidebar: '1px 0 0 rgba(44,36,32,0.06)',
        dialog: '0 8px 24px rgba(44,36,32,0.12)',
      },
      spacing: {
        'sidebar': '240px',
      },
    },
  },
  plugins: [],
}
