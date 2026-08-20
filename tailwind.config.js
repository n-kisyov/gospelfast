/** @type {import('tailwindcss').Config} */
module.exports = {
  content: ["./web/templates/**/*.html"],
  darkMode: 'class',
  theme: {
    extend: {
      // Warm, high-readability palette (inspired by the Claude Code CLI's
      // dark theme: a warm charcoal ground instead of cool blue-grays or
      // near-black, paired with a muted clay/terracotta accent instead of
      // stock blue). Overriding the built-in `gray`/`blue` scales means
      // every existing `gray-*`/`blue-*`/`dark:*` utility across the
      // templates repaints automatically — no class renames needed.
      // Every text/background pairing below is verified >=4.5:1 contrast
      // (WCAG AA), see internal/dev notes.
      colors: {
        gray: {
          50:  '#F8F8F6',
          100: '#F3F1EF',
          200: '#E7E4DF',
          300: '#D3CEC5',
          400: '#B4AC9C',
          500: '#7D725E',
          600: '#746A58',
          700: '#574F42',
          800: '#3D382E',
          900: '#2C2821',
          950: '#1A1814',
        },
        blue: {
          50:  '#FCF4F1',
          100: '#F7E6DE',
          200: '#EFC9B9',
          300: '#E3A387',
          400: '#DD906E',
          500: '#D26D41',
          600: '#BA562C',
          700: '#954523',
          800: '#74361B',
          900: '#572814',
          950: '#36190D',
        },
      },
      fontFamily: {
        sans: ['"Inter"', 'ui-sans-serif', 'system-ui', 'sans-serif'],
        serif: ['"Source Serif 4"', 'ui-serif', 'Georgia', 'serif'],
      },
    },
  },
  plugins: [],
};
