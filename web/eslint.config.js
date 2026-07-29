import next from "eslint-config-next";

const config = [
  ...next,
  {
    ignores: [
      "public/mockServiceWorker.js",
      "node_modules",
      ".next",
      "out",
      "dist",
      "*.local"
    ]
  }
];

export default config;
