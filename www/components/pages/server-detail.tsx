"use client";

import { useEffect, useRef } from "react";
import { useQuery, useMutation } from "@apollo/client";
import { Link } from "react-router-dom";
import {
  ArrowLeft,
  Pencil,
  Play,
  Square,
  Plus,
  Loader2,
  Network,
  ArrowUpDown,
  FileText,
} from "lucide-react";
import { toast } from "sonner";

import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Skeleton } from "@/components/ui/skeleton";
import { Separator } from "@/components/ui/separator";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import { CopyableText } from "@/components/copyable-text";
import { PeerClientConfigDialog } from "@/components/peers/peer-client-config-dialog";
import { formatBytes, formatDateTime, timeAgo } from "@/lib/format";
import { SERVER_QUERY } from "@/lib/graphql/queries";
import {
  START_SERVER_MUTATION,
  STOP_SERVER_MUTATION,
} from "@/lib/graphql/mutations";
import {
  SERVER_DETAIL_CHANGED_SUBSCRIPTION,
  PEER_CHANGED_SUBSCRIPTION,
} from "@/lib/graphql/subscriptions";
import type { Server, Peer } from "@/lib/graphql/types";

export default function ServerDetailPage({ id }: { id: string }) {
  const { data, loading, error, subscribeToMore } = useQuery(SERVER_QUERY, {
    variables: { id },
  });

  // Subscribe to real-time updates for this server
  const serverSubRef = useRef(false);
  useEffect(() => {
    if (serverSubRef.current) return;
    serverSubRef.current = true;

    const unsubServer = subscribeToMore({
      document: SERVER_DETAIL_CHANGED_SUBSCRIPTION,
      updateQuery: (prev, { subscriptionData }) => {
        if (!subscriptionData.data) return prev;
        const { action, node } = subscriptionData.data.serverChanged;
        const normalizedAction = String(action).toUpperCase();
        // Only update if this is the server we're viewing
        if (node.id !== id) return prev;
        if (normalizedAction === "DELETED") {
          return { ...prev, node: null };
        }
        return { ...prev, node: { ...prev.node, ...node } };
      },
    });

    const unsubPeer = subscribeToMore({
      document: PEER_CHANGED_SUBSCRIPTION,
      updateQuery: (prev, { subscriptionData }) => {
        if (!subscriptionData.data) return prev;
        const { action, node: peerNode } = subscriptionData.data.peerChanged;
        const normalizedAction = String(action).toUpperCase();
        const server = prev.node;
        if (!server || peerNode.server?.id !== id) return prev;
        const peers: Peer[] = server.peers ?? [];

        switch (normalizedAction) {
          case "CREATED":
            if (peers.some((p: Peer) => p.id === peerNode.id)) return prev;
            return { ...prev, node: { ...server, peers: [...peers, peerNode] } };
          case "UPDATED":
            return {
              ...prev,
              node: {
                ...server,
                peers: peers.map((p: Peer) =>
                  p.id === peerNode.id ? { ...p, ...peerNode } : p
                ),
              },
            };
          case "DELETED":
            return {
              ...prev,
              node: {
                ...server,
                peers: peers.filter((p: Peer) => p.id !== peerNode.id),
              },
            };
          default:
            return prev;
        }
      },
    });

    return () => {
      unsubServer();
      unsubPeer();
    };
  }, [subscribeToMore, id]);

  const server: Server | null = data?.node ?? null;

  const [startServer, { loading: starting }] = useMutation(
    START_SERVER_MUTATION,
    {
      variables: { input: { id } },
      refetchQueries: [{ query: SERVER_QUERY, variables: { id } }],
    }
  );

  const [stopServer, { loading: stopping }] = useMutation(
    STOP_SERVER_MUTATION,
    {
      variables: { input: { id } },
      refetchQueries: [{ query: SERVER_QUERY, variables: { id } }],
    }
  );

  const handleStart = async () => {
    try {
      await startServer();
      toast.success("Server started");
    } catch (err) {
      toast.error(err instanceof Error ? err.message : "Failed to start server");
    }
  };

  const handleStop = async () => {
    try {
      await stopServer();
      toast.success("Server stopped");
    } catch (err) {
      toast.error(err instanceof Error ? err.message : "Failed to stop server");
    }
  };

  if (loading) {
    return (
      <div className="flex flex-col gap-6">
        <Skeleton className="h-8 w-64" />
        <Skeleton className="h-48 w-full" />
        <Skeleton className="h-64 w-full" />
      </div>
    );
  }

  if (error || !server) {
    return (
      <div className="flex flex-col items-center justify-center py-16">
        <p className="text-sm text-muted-foreground">
          Server not found or an error occurred.
        </p>
        <Button variant="outline" className="mt-4" asChild>
          <Link to="/servers">Back to Servers</Link>
        </Button>
      </div>
    );
  }

  const peers: Peer[] = server.peers ?? [];

  return (
    <div className="flex flex-col gap-6">
      {/* Header */}
      <div className="flex flex-col gap-4 sm:flex-row sm:items-center sm:justify-between">
        <div className="flex items-center gap-3">
          <Button variant="ghost" size="icon" className="h-8 w-8" asChild>
            <Link to="/servers">
              <ArrowLeft className="h-4 w-4" />
            </Link>
          </Button>
          <div>
            <div className="flex items-center gap-2">
              <h1 className="text-2xl font-semibold tracking-tight text-foreground">
                {server.name}
              </h1>
              {server.running ? (
                <Badge
                  variant="outline"
                  className="border-success/30 bg-success/10 text-success"
                >
                  Running
                </Badge>
              ) : (
                <Badge
                  variant="outline"
                  className="border-muted-foreground/30 text-muted-foreground"
                >
                  Stopped
                </Badge>
              )}
              {!server.enabled && (
                <Badge variant="secondary">Disabled</Badge>
              )}
            </div>
            {server.description && (
              <p className="mt-0.5 text-sm text-muted-foreground">
                {server.description}
              </p>
            )}
          </div>
        </div>
        <div className="flex items-center gap-2">
          {server.running ? (
            <Button
              variant="outline"
              size="sm"
              onClick={handleStop}
              disabled={stopping}
            >
              {stopping ? (
                <Loader2 className="mr-1.5 h-3 w-3 animate-spin" />
              ) : (
                <Square className="mr-1.5 h-3 w-3" />
              )}
              Stop
            </Button>
          ) : (
            <Button
              variant="outline"
              size="sm"
              onClick={handleStart}
              disabled={starting}
            >
              {starting ? (
                <Loader2 className="mr-1.5 h-3 w-3 animate-spin" />
              ) : (
                <Play className="mr-1.5 h-3 w-3" />
              )}
              Start
            </Button>
          )}
          <Button variant="outline" size="sm" asChild>
            <Link to={`/servers/${id}/edit`}>
              <Pencil className="mr-1.5 h-3 w-3" />
              Edit
            </Link>
          </Button>
        </div>
      </div>

      {/* Server Info */}
      <div className="grid grid-cols-1 gap-4 md:grid-cols-2 lg:grid-cols-4">
        <Card>
          <CardContent className="pt-6">
            <p className="text-xs font-medium uppercase tracking-wider text-muted-foreground">
              Address
            </p>
            <p className="mt-1 font-mono text-sm text-foreground">
              {server.address}
            </p>
          </CardContent>
        </Card>
        <Card>
          <CardContent className="pt-6">
            <p className="text-xs font-medium uppercase tracking-wider text-muted-foreground">
              Listen Port
            </p>
            <p className="mt-1 font-mono text-sm text-foreground">
              {server.listenPort ?? "Auto"}
            </p>
          </CardContent>
        </Card>
        <Card>
          <CardContent className="pt-6">
            <p className="text-xs font-medium uppercase tracking-wider text-muted-foreground">
              MTU
            </p>
            <p className="mt-1 font-mono text-sm text-foreground">
              {server.mtu}
            </p>
          </CardContent>
        </Card>
        <Card>
          <CardContent className="pt-6">
            <p className="text-xs font-medium uppercase tracking-wider text-muted-foreground">
              Peers
            </p>
            <p className="mt-1 text-sm text-foreground">{peers.length}</p>
          </CardContent>
        </Card>
      </div>

      {/* Extra Info */}
      <Card>
        <CardContent className="flex flex-col gap-4 pt-6">
          <div className="flex flex-col gap-2 sm:flex-row sm:items-center sm:gap-6">
            <div>
              <span className="text-xs text-muted-foreground">Public Key</span>
              <CopyableText text={server.publicKey} truncate={false} />
            </div>
          </div>
          {server.dns && server.dns.length > 0 && (
            <div>
              <span className="text-xs text-muted-foreground">DNS</span>
              <p className="font-mono text-sm text-foreground">
                {server.dns.join(", ")}
              </p>
            </div>
          )}
          {server.interfaceStats && (
            <div className="flex items-center gap-6">
              <div>
                <span className="text-xs text-muted-foreground">RX</span>
                <p className="text-sm text-foreground">
                  {formatBytes(server.interfaceStats.rxBytes)}
                </p>
              </div>
              <div>
                <span className="text-xs text-muted-foreground">TX</span>
                <p className="text-sm text-foreground">
                  {formatBytes(server.interfaceStats.txBytes)}
                </p>
              </div>
            </div>
          )}
          <div className="flex items-center gap-4 text-xs text-muted-foreground">
            <span>Created {formatDateTime(server.createdAt)}</span>
            <span>Updated {formatDateTime(server.updatedAt)}</span>
          </div>
        </CardContent>
      </Card>

      {/* Peers Table */}
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-2">
          <Network className="h-4 w-4 text-muted-foreground" />
          <h2 className="text-lg font-semibold text-foreground">Peers</h2>
          <Badge variant="secondary">{peers.length}</Badge>
        </div>
        <Button size="sm" asChild>
          <Link to={`/servers/${id}/peers/new`}>
            <Plus className="mr-1.5 h-3 w-3" />
            Add Peer
          </Link>
        </Button>
      </div>

      {peers.length === 0 ? (
        <div className="flex flex-col items-center justify-center rounded-lg border border-dashed border-border py-12">
          <Network className="mb-3 h-8 w-8 text-muted-foreground/50" />
          <p className="text-sm font-medium text-foreground">No peers yet</p>
          <p className="mt-1 text-sm text-muted-foreground">
            Add a peer to allow connections to this server.
          </p>
          <Button size="sm" className="mt-4" asChild>
            <Link to={`/servers/${id}/peers/new`}>
              <Plus className="mr-1.5 h-3 w-3" />
              Add Peer
            </Link>
          </Button>
        </div>
      ) : (
        <div className="rounded-lg border border-border">
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>Name</TableHead>
                <TableHead className="hidden md:table-cell">
                  Public Key
                </TableHead>
                <TableHead className="hidden md:table-cell">
                  Endpoint
                </TableHead>
                <TableHead className="hidden lg:table-cell">
                  Allowed IPs
                </TableHead>
                <TableHead className="hidden lg:table-cell">
                  Handshake
                </TableHead>
                <TableHead className="hidden lg:table-cell">Traffic</TableHead>
                <TableHead className="w-12">
                  <span className="sr-only">Actions</span>
                </TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {peers.map((peer) => (
                <TableRow key={peer.id}>
                  <TableCell>
                    <span className="font-medium text-foreground">
                      {peer.name}
                    </span>
                    {peer.description && (
                      <p className="mt-0.5 max-w-xs truncate text-xs text-muted-foreground">
                        {peer.description}
                      </p>
                    )}
                  </TableCell>
                  <TableCell className="hidden md:table-cell">
                    <code className="max-w-[150px] truncate rounded bg-muted px-1.5 py-0.5 font-mono text-xs text-foreground">
                      {peer.publicKey.slice(0, 16)}...
                    </code>
                  </TableCell>
                  <TableCell className="hidden font-mono text-xs text-muted-foreground md:table-cell">
                    {peer.stats?.endpoint || peer.endpoint || "--"}
                  </TableCell>
                  <TableCell className="hidden text-xs text-muted-foreground lg:table-cell">
                    {peer.allowedIPs?.join(", ") || "--"}
                  </TableCell>
                  <TableCell className="hidden text-xs text-muted-foreground lg:table-cell">
                    {peer.stats?.lastHandshakeTime
                      ? timeAgo(peer.stats.lastHandshakeTime)
                      : "Never"}
                  </TableCell>
                  <TableCell className="hidden lg:table-cell">
                    {peer.stats ? (
                      <div className="flex items-center gap-1 text-xs text-muted-foreground">
                        <ArrowUpDown className="h-3 w-3" />
                        {formatBytes(peer.stats.receiveBytes)} /{" "}
                        {formatBytes(peer.stats.transmitBytes)}
                      </div>
                    ) : (
                      <span className="text-xs text-muted-foreground">--</span>
                    )}
                  </TableCell>
                  <TableCell>
                    <div className="flex items-center gap-1">
                      <PeerClientConfigDialog
                        server={server}
                        peer={peer}
                        trigger={
                          <Button variant="ghost" size="sm">
                            <FileText className="h-3 w-3" />
                          </Button>
                        }
                      />
                      <Button variant="ghost" size="sm" asChild>
                        <Link to={`/servers/${id}/peers/${peer.id}/edit`}>
                          <Pencil className="h-3 w-3" />
                        </Link>
                      </Button>
                    </div>
                  </TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        </div>
      )}
    </div>
  );
}
