const defaultTheme = require("tailwindcss/defaultTheme");

/** @type {import('tailwindcss').Config} */
module.exports = {
  content: ["./src/**/*.{astro,html,js,jsx,md,mdx,svelte,ts,tsx,vue}"],
  theme: {
    extend: {
      fontFamily: {
        body: ["Archivo", ...defaultTheme.fontFamily.sans],
        display: ["Sora", ...defaultTheme.fontFamily.serif],
      },
      colors: {
        blue: {
          DEFAULT: "#2C40F7",
          50: "#F3F4FF",
          100: "#DDE0FE",
          200: "#B1B8FC",
          300: "#8490FA",
          400: "#5868F9",
          500: "#2C40F7",
          600: "#0718B6",
          800: "#030C59",
          900: "#010319",
        },
      },
      maxWidth: {
        "7xl": "90rem", // Set your desired max width value here
      },
    },
  },
  plugins: [require("@tailwindcss/forms"), require("tailwindcss-animate")],
};
