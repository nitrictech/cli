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
      maxWidth: {
        "7xl": "90rem", // Set your desired max width value here
      },
    },
  },
  plugins: [require("@tailwindcss/forms")],
};
