"use client";

import { useQuery } from "@apollo/client";
import { useNavigate } from "react-router-dom";
import { useEffect, createContext, useContext } from "react";
import { VIEWER_QUERY } from "@/lib/graphql/queries";
import { isAuthenticated } from "@/lib/auth";
import type { User } from "@/lib/graphql/types";
import { Skeleton } from "@/components/ui/skeleton";

interface AuthContextValue {
  user: User;
}

const AuthContext = createContext<AuthContextValue | null>(null);

export function useAuth() {
  const ctx = useContext(AuthContext);
  if (!ctx) throw new Error("useAuth must be used within AuthGuard");
  return ctx;
}

export function AuthGuard({ children }: { children: React.ReactNode }) {
  const navigate = useNavigate();
  const hasToken = isAuthenticated();
  const { data, loading, error } = useQuery(VIEWER_QUERY, {
    fetchPolicy: "cache-first",
    skip: !hasToken,
  });

  useEffect(() => {
    if (!hasToken) {
      navigate("/login", { replace: true });
      return;
    }
    if (!loading && (error || !data?.viewer)) {
      navigate("/login", { replace: true });
    }
  }, [hasToken, loading, error, data, navigate]);

  // No token at all - will redirect in useEffect
  if (!hasToken) {
    return null;
  }

  // Still loading the initial viewer query (no cached data yet)
  if (loading && !data?.viewer) {
    return (
      <div className="flex min-h-screen flex-col">
        <div className="flex h-14 items-center border-b px-6">
          <Skeleton className="h-5 w-40" />
          <div className="ml-auto flex items-center gap-4">
            <Skeleton className="h-5 w-20" />
            <Skeleton className="h-5 w-20" />
            <Skeleton className="h-5 w-20" />
          </div>
        </div>
        <div className="flex-1 p-6">
          <Skeleton className="mb-4 h-8 w-48" />
          <Skeleton className="h-64 w-full" />
        </div>
      </div>
    );
  }

  if (!data?.viewer) {
    return null;
  }

  return (
    <AuthContext.Provider value={{ user: data.viewer }}>
      {children}
    </AuthContext.Provider>
  );
}
