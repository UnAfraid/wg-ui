"use client";

import { useState, useEffect } from "react";
import dynamic from "next/dynamic";

const AppShell = dynamic(() => import("@/components/app-shell"), {
  ssr: false,
});

export default function Page() {
  const [mounted, setMounted] = useState(false);
  useEffect(() => setMounted(true), []);
  if (!mounted) return null;
  return <AppShell />;
}
