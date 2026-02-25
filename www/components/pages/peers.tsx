"use client";

import { Fragment, useEffect, useMemo, useRef, useState } from "react";
import { useQuery } from "@apollo/client";
import { Link } from "react-router-dom";
import { Search, Loader2, Network, Pencil } from "lucide-react";

import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import { Badge } from "@/components/ui/badge";
import { CopyableText } from "@/components/copyable-text";
import { PEERS_QUERY } from "@/lib/graphql/queries";
import { PEER_CHANGED_SUBSCRIPTION } from "@/lib/graphql/subscriptions";
import type { Peer } from "@/lib/graphql/types";

type GroupBy = "none" | "server" | "backend";

interface PeerGroup {
  key: string;
  label: string;
  peers: Peer[];
}

export default function PeersPage() {
  const [search, setSearch] = useState("");
  const [groupBy, setGroupBy] = useState<GroupBy>("none");
  const { data, loading, subscribeToMore } = useQuery(PEERS_QUERY);

  const subscribedRef = useRef(false);
  useEffect(() => {
    if (subscribedRef.current) return;
    subscribedRef.current = true;

    const unsubscribe = subscribeToMore({
      document: PEER_CHANGED_SUBSCRIPTION,
      updateQuery: (prev, { subscriptionData }) => {
        if (!subscriptionData.data) return prev;
        const { action, node } = subscriptionData.data.peerChanged;
        const normalizedAction = String(action).toUpperCase();
        const existing: Peer[] = prev.peers ?? [];

        switch (normalizedAction) {
          case "CREATED":
            if (existing.some((p) => p.id === node.id)) return prev;
            return { ...prev, peers: [...existing, node] };
          case "UPDATED":
            return {
              ...prev,
              peers: existing.map((p) => (p.id === node.id ? { ...p, ...node } : p)),
            };
          case "DELETED":
            return { ...prev, peers: existing.filter((p) => p.id !== node.id) };
          default:
            return prev;
        }
      },
    });

    return () => unsubscribe();
  }, [subscribeToMore]);

  const peers: Peer[] = data?.peers ?? [];
  const searchValue = search.trim().toLowerCase();

  const filteredPeers = useMemo(() => {
    const list = searchValue
      ? peers.filter((peer) => {
          const name = (peer.name ?? "").toLowerCase();
          const publicKey = (peer.publicKey ?? "").toLowerCase();
          return name.includes(searchValue) || publicKey.includes(searchValue);
        })
      : peers;

    return [...list].sort((a, b) =>
      a.name.localeCompare(b.name, undefined, { sensitivity: "base" })
    );
  }, [peers, searchValue]);

  const groupedPeers = useMemo<PeerGroup[]>(() => {
    if (groupBy === "none") {
      return [{ key: "all", label: "All peers", peers: filteredPeers }];
    }

    const groups = new Map<string, PeerGroup>();
    for (const peer of filteredPeers) {
      const groupName =
        groupBy === "server"
          ? peer.server?.name?.trim() || "No Server"
          : peer.backend?.name?.trim() || "No Backend";
      const groupId =
        groupBy === "server"
          ? peer.server?.id?.trim()
          : peer.backend?.id?.trim();
      const groupKey = groupId || `name:${groupName.toLowerCase()}`;

      let group = groups.get(groupKey);
      if (!group) {
        group = { key: groupKey, label: groupName, peers: [] };
        groups.set(groupKey, group);
      }
      group.peers.push(peer);
    }

    const result = Array.from(groups.values());
    result.sort((a, b) =>
      a.label.localeCompare(b.label, undefined, { sensitivity: "base" })
    );
    for (const group of result) {
      group.peers.sort((a, b) =>
        a.name.localeCompare(b.name, undefined, { sensitivity: "base" })
      );
    }

    return result;
  }, [filteredPeers, groupBy]);

  return (
    <div className="flex flex-col gap-6">
      <div>
        <h1 className="text-2xl font-semibold tracking-tight text-foreground">
          Peers
        </h1>
        <p className="mt-1 text-sm text-muted-foreground">
          Search peers by name or public key across all servers.
        </p>
      </div>

      {peers.length > 0 && (
        <div className="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
          <div className="relative w-full max-w-md">
            <Search className="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground" />
            <Input
              placeholder="Search by peer name or public key..."
              value={search}
              onChange={(e) => setSearch(e.target.value)}
              className="pl-9"
            />
          </div>

          <div className="w-full sm:w-48">
            <Select
              value={groupBy}
              onValueChange={(value) => setGroupBy(value as GroupBy)}
            >
              <SelectTrigger>
                <SelectValue placeholder="Group by" />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="none">Group: None</SelectItem>
                <SelectItem value="server">Group: Server</SelectItem>
                <SelectItem value="backend">Group: Backend</SelectItem>
              </SelectContent>
            </Select>
          </div>
        </div>
      )}

      {loading ? (
        <div className="flex items-center justify-center py-16">
          <Loader2 className="h-6 w-6 animate-spin text-muted-foreground" />
        </div>
      ) : filteredPeers.length === 0 ? (
        <div className="flex flex-col items-center justify-center rounded-lg border border-dashed border-border py-16">
          <Network className="mb-3 h-10 w-10 text-muted-foreground/50" />
          <p className="text-sm font-medium text-foreground">No peers found</p>
          <p className="mt-1 text-sm text-muted-foreground">
            {searchValue ? "Try a different search term." : "No peers have been created yet."}
          </p>
        </div>
      ) : (
        <div className="rounded-lg border border-border">
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>Name</TableHead>
                <TableHead className="hidden lg:table-cell">Public Key</TableHead>
                <TableHead>Server</TableHead>
                <TableHead>Backend</TableHead>
                <TableHead className="hidden md:table-cell">Endpoint</TableHead>
                <TableHead className="w-12">
                  <span className="sr-only">Actions</span>
                </TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {groupedPeers.map((group) => (
                <Fragment key={group.key}>
                  {groupBy !== "none" && (
                    <TableRow className="bg-muted/20 hover:bg-muted/20">
                      <TableCell colSpan={6} className="py-2">
                        <div className="flex items-center justify-between gap-2">
                          <span className="text-xs font-semibold uppercase tracking-wide text-muted-foreground">
                            {group.label}
                          </span>
                          <Badge variant="secondary" className="text-[10px]">
                            {group.peers.length}
                          </Badge>
                        </div>
                      </TableCell>
                    </TableRow>
                  )}
                  {group.peers.map((peer) => {
                    const serverDetailPath = peer.server?.id
                      ? `/servers/${encodeURIComponent(peer.server.id)}`
                      : null;
                    const peerEditPath =
                      serverDetailPath != null
                        ? `${serverDetailPath}/peers/${encodeURIComponent(peer.id)}/edit`
                        : null;

                    return (
                      <TableRow key={peer.id}>
                        <TableCell>
                          <span className="font-medium text-foreground">{peer.name}</span>
                          {peer.description && (
                            <p className="mt-0.5 max-w-xs truncate text-xs text-muted-foreground">
                              {peer.description}
                            </p>
                          )}
                        </TableCell>
                        <TableCell className="hidden lg:table-cell">
                          <CopyableText text={peer.publicKey} />
                        </TableCell>
                        <TableCell>
                          {serverDetailPath ? (
                            <Link
                              to={serverDetailPath}
                              className="text-sm font-medium text-foreground hover:text-primary"
                            >
                              {peer.server.name}
                            </Link>
                          ) : (
                            <span className="text-sm text-muted-foreground">--</span>
                          )}
                        </TableCell>
                        <TableCell>
                          {peer.backend?.name ? (
                            <Badge variant="secondary">{peer.backend.name}</Badge>
                          ) : (
                            <span className="text-sm text-muted-foreground">--</span>
                          )}
                        </TableCell>
                        <TableCell className="hidden font-mono text-xs text-muted-foreground md:table-cell">
                          {peer.stats?.endpoint || peer.endpoint || "--"}
                        </TableCell>
                        <TableCell>
                          {peerEditPath ? (
                            <Button variant="ghost" size="sm" asChild>
                              <Link to={peerEditPath}>
                                <Pencil className="h-3 w-3" />
                              </Link>
                            </Button>
                          ) : (
                            <Button variant="ghost" size="sm" disabled>
                              <Pencil className="h-3 w-3" />
                            </Button>
                          )}
                        </TableCell>
                      </TableRow>
                    );
                  })}
                </Fragment>
              ))}
            </TableBody>
          </Table>
        </div>
      )}
    </div>
  );
}
