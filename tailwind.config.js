/** @type {import('tailwindcss').Config} */
const colors = require('tailwindcss/colors')
module.exports = {
  content: ["./templates/**/*.{html,tmpl}"],
  theme: {
    extend: {
      colors: {
        accent: colors.violet,
      },
    }
  },
  plugins: [],
}

