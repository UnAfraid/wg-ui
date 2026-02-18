"use client";

import { useState, useEffect, useRef } from "react";
import { useQuery } from "@apollo/client";
import { Link, useSearchParams } from "react-router-dom";
import { Plus, Search, Loader2, X } from "lucide-react";

import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Badge } from "@/components/ui/badge";
import { ServerTable } from "@/components/servers/server-table";
import { ForeignServers } from "@/components/servers/foreign-servers";
import { SERVERS_QUERY, BACKENDS_QUERY } from "@/lib/graphql/queries";
import { SERVER_CHANGED_SUBSCRIPTION } from "@/lib/graphql/subscriptions";
import type { Server, Backend } from "@/lib/graphql/types";

export default function ServersPage() {
  const [search, setSearch] = useState("");
  const [searchParams, setSearchParams] = useSearchParams();
  const backendFilter = searchParams.get("backend");
  const { data, loading, subscribeToMore } = useQuery(SERVERS_QUERY);
  const { data: backendsData } = useQuery(BACKENDS_QUERY);

  // Subscribe to real-time server changes
  const subscribedRef = useRef(false);
  useEffect(() => {
    if (subscribedRef.current) return;
    subscribedRef.current = true;

    const unsubscribe = subscribeToMore({
      document: SERVER_CHANGED_SUBSCRIPTION,
      updateQuery: (prev, { subscriptionData }) => {
        if (!subscriptionData.data) return prev;
        const { action, node } = subscriptionData.data.serverChanged;
        const normalizedAction = String(action).toUpperCase();
        const existing: Server[] = prev.servers ?? [];

        switch (normalizedAction) {
          case "CREATED":
            // Add if not already in list
            if (existing.some((s: Server) => s.id === node.id)) return prev;
            return { ...prev, servers: [...existing, node] };

          case "UPDATED":
          case "STARTED":
          case "STOPPED":
          case "INTERFACE_STATS_UPDATED":
            return {
              ...prev,
              servers: existing.map((s: Server) =>
                s.id === node.id ? { ...s, ...node } : s
              ),
            };

          case "DELETED":
            return {
              ...prev,
              servers: existing.filter((s: Server) => s.id !== node.id),
            };

          default:
            return prev;
        }
      },
    });

    return () => unsubscribe();
  }, [subscribeToMore]);

  const servers: Server[] = data?.servers ?? [];
  const backends: Backend[] = backendsData?.backends ?? [];
  const backendName = backends.find((b) => b.id === backendFilter)?.name;

  const filtered = servers.filter((s) => {
    const matchesBackend = !backendFilter || s.backend?.id === backendFilter;
    const matchesSearch =
      !search ||
      s.name.toLowerCase().includes(search.toLowerCase()) ||
      s.address.toLowerCase().includes(search.toLowerCase());
    return matchesBackend && matchesSearch;
  });

  const clearBackendFilter = () => {
    searchParams.delete("backend");
    setSearchParams(searchParams);
  };

  return (
    <div className="flex flex-col gap-6">
      <div className="flex flex-col gap-4 sm:flex-row sm:items-center sm:justify-between">
        <div>
          <h1 className="text-2xl font-semibold tracking-tight text-foreground">
            Servers
          </h1>
          <p className="mt-1 text-sm text-muted-foreground">
            Manage your WireGuard server interfaces
          </p>
        </div>
        <Button asChild>
          <Link to="/servers/new">
            <Plus className="mr-1.5 h-4 w-4" />
            New Server
          </Link>
        </Button>
      </div>

      <div className="flex flex-wrap items-center gap-3">
        {servers.length > 0 && (
          <div className="relative max-w-sm">
            <Search className="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground" />
            <Input
              placeholder="Search servers..."
              value={search}
              onChange={(e) => setSearch(e.target.value)}
              className="pl-9"
            />
          </div>
        )}
        {backendFilter && (
          <Badge variant="secondary" className="gap-1.5 px-3 py-1.5 text-sm">
            Backend: {backendName ?? backendFilter}
            <button
              type="button"
              onClick={clearBackendFilter}
              className="ml-0.5 hover:text-destructive"
              aria-label="Clear backend filter"
            >
              <X className="h-3 w-3" />
            </button>
          </Badge>
        )}
      </div>

      {loading ? (
        <div className="flex items-center justify-center py-16">
          <Loader2 className="h-6 w-6 animate-spin text-muted-foreground" />
        </div>
      ) : (
        <ServerTable servers={filtered} />
      )}

      <ForeignServers />
    </div>
  );
}
