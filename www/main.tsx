import { StrictMode } from "react";
import { createRoot } from "react-dom/client";
import { ThemeProvider } from "@/components/theme-provider";
import { ApolloWrapper } from "@/lib/apollo-provider";
import { Toaster } from "@/components/ui/sonner";
import AppShell from "@/components/app-shell";

import "./globals.css";

createRoot(document.getElementById("root")!).render(
  <StrictMode>
    <ThemeProvider
      attribute="class"
      defaultTheme="dark"
      enableSystem
      disableTransitionOnChange
    >
      <ApolloWrapper>
        <AppShell />
        <Toaster richColors position="bottom-right" />
      </ApolloWrapper>
    </ThemeProvider>
  </StrictMode>,
);
