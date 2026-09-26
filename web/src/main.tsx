import React from "react";
import { createRoot } from "react-dom/client";

function App() {
  return (
    <main>
      <h1>AIDI 2.0</h1>
      <p>Release A — Foundation is active.</p>
    </main>
  );
}

const root = document.getElementById("root");
if (!root) {
  throw new Error("root element not found");
}

createRoot(root).render(
  <React.StrictMode>
    <App />
  </React.StrictMode>,
);
