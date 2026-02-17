"use client";

import { useState, useMemo } from "react";
import { Link } from "react-router-dom";
import { useMutation } from "@apollo/client";
import {
  Play,
  Square,
  Pencil,
  Trash2,
  MoreHorizontal,
  ArrowUpDown,
  ArrowDownUp,
  ChevronUp,
  ChevronDown,
} from "lucide-react";
import { toast } from "sonner";

import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { ConfirmDialog } from "@/components/confirm-dialog";
import { CopyableText } from "@/components/copyable-text";
import { formatBytes } from "@/lib/format";
import {
  START_SERVER_MUTATION,
  STOP_SERVER_MUTATION,
  DELETE_SERVER_MUTATION,
} from "@/lib/graphql/mutations";
import { SERVERS_QUERY } from "@/lib/graphql/queries";
import type { Server } from "@/lib/graphql/types";

interface ServerTableProps {
  servers: Server[];
}

type SortKey = "name" | "backend" | "address" | "status" | "peers";
type SortDir = "asc" | "desc";

function SortableHead({
  label,
  sortKey,
  currentKey,
  currentDir,
  onSort,
  className,
}: {
  label: string;
  sortKey: SortKey;
  currentKey: SortKey;
  currentDir: SortDir;
  onSort: (key: SortKey) => void;
  className?: string;
}) {
  const active = currentKey === sortKey;
  return (
    <TableHead className={className}>
      <button
        type="button"
        onClick={() => onSort(sortKey)}
        className="inline-flex items-center gap-1 text-xs hover:text-foreground"
      >
        {label}
        {active ? (
          currentDir === "asc" ? (
            <ChevronUp className="h-3.5 w-3.5" />
          ) : (
            <ChevronDown className="h-3.5 w-3.5" />
          )
        ) : (
          <ArrowUpDown className="h-3 w-3 text-muted-foreground/50" />
        )}
      </button>
    </TableHead>
  );
}

export function ServerTable({ servers }: ServerTableProps) {
  const [sortKey, setSortKey] = useState<SortKey>("name");
  const [sortDir, setSortDir] = useState<SortDir>("asc");

  const handleSort = (key: SortKey) => {
    if (key === sortKey) {
      setSortDir((d) => (d === "asc" ? "desc" : "asc"));
    } else {
      setSortKey(key);
      setSortDir("asc");
    }
  };

  const sorted = useMemo(() => {
    const copy = [...servers];
    const dir = sortDir === "asc" ? 1 : -1;
    copy.sort((a, b) => {
      switch (sortKey) {
        case "name":
          return dir * a.name.localeCompare(b.name);
        case "backend":
          return dir * (a.backend?.name ?? "").localeCompare(b.backend?.name ?? "");
        case "address":
          return dir * a.address.localeCompare(b.address);
        case "status": {
          const aVal = a.running ? 1 : 0;
          const bVal = b.running ? 1 : 0;
          return dir * (aVal - bVal);
        }
        case "peers":
          return dir * ((a.peers?.length ?? 0) - (b.peers?.length ?? 0));
        default:
          return 0;
      }
    });
    return copy;
  }, [servers, sortKey, sortDir]);

  const [startServer] = useMutation(START_SERVER_MUTATION, {
    refetchQueries: [{ query: SERVERS_QUERY }],
  });

  const [stopServer] = useMutation(STOP_SERVER_MUTATION, {
    refetchQueries: [{ query: SERVERS_QUERY }],
  });

  const [deleteServer] = useMutation(DELETE_SERVER_MUTATION, {
    refetchQueries: [{ query: SERVERS_QUERY }],
  });

  const handleStart = async (id: string) => {
    try {
      await startServer({ variables: { input: { id } } });
      toast.success("Server started");
    } catch (err) {
      toast.error(err instanceof Error ? err.message : "Failed to start");
    }
  };

  const handleStop = async (id: string) => {
    try {
      await stopServer({ variables: { input: { id } } });
      toast.success("Server stopped");
    } catch (err) {
      toast.error(err instanceof Error ? err.message : "Failed to stop");
    }
  };

  const handleDelete = async (id: string) => {
    try {
      await deleteServer({ variables: { input: { id } } });
      toast.success("Server deleted");
    } catch (err) {
      toast.error(err instanceof Error ? err.message : "Failed to delete");
    }
  };

  if (servers.length === 0) {
    return (
      <div className="flex flex-col items-center justify-center rounded-lg border border-dashed border-border py-16">
        <ArrowDownUp className="mb-3 h-10 w-10 text-muted-foreground/50" />
        <p className="text-sm font-medium text-foreground">No servers yet</p>
        <p className="mt-1 text-sm text-muted-foreground">
          Create your first WireGuard server to get started.
        </p>
      </div>
    );
  }

  return (
    <div className="rounded-lg border border-border">
      <Table>
        <TableHeader>
          <TableRow>
            <SortableHead label="Name" sortKey="name" currentKey={sortKey} currentDir={sortDir} onSort={handleSort} />
            <SortableHead label="Backend" sortKey="backend" currentKey={sortKey} currentDir={sortDir} onSort={handleSort} className="hidden lg:table-cell" />
            <SortableHead label="Address" sortKey="address" currentKey={sortKey} currentDir={sortDir} onSort={handleSort} />
            <TableHead className="hidden md:table-cell">Port</TableHead>
            <SortableHead label="Status" sortKey="status" currentKey={sortKey} currentDir={sortDir} onSort={handleSort} />
            <SortableHead label="Peers" sortKey="peers" currentKey={sortKey} currentDir={sortDir} onSort={handleSort} className="hidden md:table-cell" />
            <TableHead className="hidden lg:table-cell">Traffic</TableHead>
            <TableHead className="w-12">
              <span className="sr-only">Actions</span>
            </TableHead>
          </TableRow>
        </TableHeader>
        <TableBody>
          {sorted.map((server) => (
            <TableRow key={server.id}>
              <TableCell>
                <Link
                  to={`/servers/${server.id}`}
                  className="font-medium text-foreground hover:text-primary"
                >
                  {server.name}
                </Link>
                {server.description && (
                  <p className="mt-0.5 max-w-xs truncate text-xs text-muted-foreground">
                    {server.description}
                  </p>
                )}
              </TableCell>
              <TableCell className="hidden lg:table-cell">
                <span className="text-sm text-muted-foreground">
                  {server.backend?.name ?? "--"}
                </span>
              </TableCell>
              <TableCell>
                <CopyableText text={server.address} truncate={false} />
              </TableCell>
              <TableCell className="hidden md:table-cell">
                <span className="font-mono text-sm text-muted-foreground">
                  {server.listenPort ?? "Auto"}
                </span>
              </TableCell>
              <TableCell>
                <div className="flex items-center gap-2">
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
                    <Badge variant="secondary" className="text-xs">
                      Disabled
                    </Badge>
                  )}
                </div>
              </TableCell>
              <TableCell className="hidden md:table-cell">
                <span className="text-sm text-muted-foreground">
                  {server.peers?.length ?? 0}
                </span>
              </TableCell>
              <TableCell className="hidden lg:table-cell">
                {server.interfaceStats ? (
                  <div className="flex items-center gap-2 text-xs text-muted-foreground">
                    <span className="flex items-center gap-1">
                      <ArrowUpDown className="h-3 w-3" />
                      {formatBytes(server.interfaceStats.rxBytes)} /{" "}
                      {formatBytes(server.interfaceStats.txBytes)}
                    </span>
                  </div>
                ) : (
                  <span className="text-xs text-muted-foreground">--</span>
                )}
              </TableCell>
              <TableCell>
                <DropdownMenu>
                  <DropdownMenuTrigger asChild>
                    <Button variant="ghost" size="icon" className="h-8 w-8">
                      <MoreHorizontal className="h-4 w-4" />
                      <span className="sr-only">Actions</span>
                    </Button>
                  </DropdownMenuTrigger>
                  <DropdownMenuContent align="end">
                    {server.running ? (
                      <DropdownMenuItem
                        onClick={() => handleStop(server.id)}
                      >
                        <Square className="mr-2 h-4 w-4" />
                        Stop
                      </DropdownMenuItem>
                    ) : (
                      <DropdownMenuItem
                        onClick={() => handleStart(server.id)}
                      >
                        <Play className="mr-2 h-4 w-4" />
                        Start
                      </DropdownMenuItem>
                    )}
                    <DropdownMenuItem asChild>
                      <Link to={`/servers/${server.id}/edit`}>
                        <Pencil className="mr-2 h-4 w-4" />
                        Edit
                      </Link>
                    </DropdownMenuItem>
                    <DropdownMenuSeparator />
                    <ConfirmDialog
                      trigger={
                        <DropdownMenuItem
                          onSelect={(e) => e.preventDefault()}
                          className="text-destructive focus:text-destructive"
                        >
                          <Trash2 className="mr-2 h-4 w-4" />
                          Delete
                        </DropdownMenuItem>
                      }
                      title="Delete Server"
                      description={`Are you sure you want to delete "${server.name}"? This will also remove all associated peers. This action cannot be undone.`}
                      confirmLabel="Delete Server"
                      onConfirm={() => handleDelete(server.id)}
                    />
                  </DropdownMenuContent>
                </DropdownMenu>
              </TableCell>
            </TableRow>
          ))}
        </TableBody>
      </Table>
    </div>
  );
}
