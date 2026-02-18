"use client";

import { useState, useMemo } from "react";
import { useQuery, useMutation } from "@apollo/client";
import { Link, useNavigate } from "react-router-dom";
import {
  Plus,
  Search,
  Loader2,
  HardDrive,
  Pencil,
  Trash2,
  MoreHorizontal,
  ArrowUpDown,
  ChevronUp,
  ChevronDown,
  ExternalLink,
} from "lucide-react";
import { toast } from "sonner";

import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import { Badge } from "@/components/ui/badge";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from "@/components/ui/dialog";
import { Label } from "@/components/ui/label";
import { Switch } from "@/components/ui/switch";
import { Checkbox } from "@/components/ui/checkbox";
import { ConfirmDialog } from "@/components/confirm-dialog";
import { BACKENDS_QUERY, AVAILABLE_BACKENDS_QUERY } from "@/lib/graphql/queries";
import {
  CREATE_BACKEND_MUTATION,
  UPDATE_BACKEND_MUTATION,
  DELETE_BACKEND_MUTATION,
} from "@/lib/graphql/mutations";
import type { Backend, AvailableBackend } from "@/lib/graphql/types";

type SortKey = "name" | "url" | "servers" | "status";
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

// ─── URL builder types & helpers ────────────────────────────────────────────

type BackendType = string;

const backendTypeMeta: Record<string, { label: string; description: string; hasFields: boolean }> = {
  linux: { label: "Linux", description: "Local Linux netlink / wgctrl", hasFields: false },
  darwin: { label: "macOS", description: "Local macOS backend", hasFields: false },
  networkmanager: { label: "NetworkManager", description: "NetworkManager via D-Bus", hasFields: false },
  exec: { label: "Exec (wg-quick)", description: "Run wg-quick or similar tools locally", hasFields: true },
  routeros: { label: "RouterOS", description: "Manage MikroTik WireGuard via RouterOS API", hasFields: true },
  ssh: { label: "SSH (remote)", description: "Manage WireGuard on remote hosts via SSH", hasFields: true },
};

function getTypeMeta(type: string) {
  return backendTypeMeta[type] ?? { label: type, description: `${type} backend`, hasFields: false };
}

interface UrlParts {
  type: BackendType;
  host: string;
  port: string;
  user: string;
  password: string;
  path: string;
  sudo: boolean;
  insecureSkipVerify: boolean;
}

function parseBoolParam(value: string | null, fallback: boolean): boolean {
  if (value == null) return fallback;
  const normalized = value.trim().toLowerCase();
  return normalized === "true" || normalized === "1" || normalized === "yes" || normalized === "on";
}

function normalizeRedactedPassword(value: string): string {
  const trimmed = value.trim();
  if (!trimmed) return "";

  let current = trimmed;
  for (let i = 0; i < 4; i++) {
    if (current === "***") return "***";
    if (current.toLowerCase() === "%2a%2a%2a") return "***";

    try {
      const decoded = decodeURIComponent(current);
      if (decoded === current) break;
      current = decoded;
    } catch {
      break;
    }
  }

  return trimmed;
}

function parseBackendUrl(raw: string): UrlParts {
  const defaults: UrlParts = {
    type: "linux",
    host: "",
    port: "22",
    user: "root",
    password: "",
    path: "/etc/wireguard",
    sudo: false,
    insecureSkipVerify: false,
  };

  if (!raw) return defaults;

  const schemeMatch = raw.match(/^([a-z]+):\/\//);
  if (!schemeMatch) return defaults;

  const scheme = schemeMatch[1];
  defaults.type = scheme;

  if (scheme === "ssh") {
    try {
      const url = new URL(raw);
      defaults.host = url.hostname;
      defaults.port = url.port || "22";
      defaults.user = url.username || "root";
      defaults.path = url.pathname || "/etc/wireguard";
      defaults.sudo = url.searchParams.get("sudo") === "true";
    } catch {
      /* invalid URL, keep defaults */
    }
  } else if (scheme === "exec") {
    try {
      const url = new URL(raw);
      defaults.path = url.pathname || "/etc/wireguard";
      defaults.sudo = url.searchParams.get("sudo") === "true";
    } catch {
      /* keep defaults */
    }
  } else if (scheme === "routeros") {
    try {
      const url = new URL(raw);
      defaults.host = url.hostname;
      defaults.port = url.port || "443";
      defaults.user = url.username || "admin";
      defaults.password = normalizeRedactedPassword(url.password || "");
      defaults.path = url.pathname || "/rest";
      defaults.insecureSkipVerify = parseBoolParam(
        url.searchParams.get("insecureSkipVerify"),
        false
      );
    } catch {
      /* keep defaults */
    }
  }

  return defaults;
}

function buildBackendUrl(parts: UrlParts): string {
  switch (parts.type) {
    case "exec": {
      const p = parts.path || "/etc/wireguard";
      const q = parts.sudo ? "?sudo=true" : "";
      return `exec://${p}${q}`;
    }
    case "ssh": {
      const host = parts.host || "hostname";
      const port = parts.port && parts.port !== "22" ? `:${parts.port}` : "";
      const user = parts.user || "root";
      const p = parts.path || "/etc/wireguard";
      const q = parts.sudo ? "?sudo=true" : "";
      return `ssh://${user}@${host}${port}${p}${q}`;
    }
    case "routeros": {
      const host = parts.host || "router";
      const portDefault = "443";
      const port = parts.port && parts.port !== portDefault ? `:${parts.port}` : "";
      const user = encodeURIComponent(parts.user || "admin");
      const password = encodeURIComponent(parts.password || "password");
      const p = parts.path
        ? parts.path.startsWith("/")
          ? parts.path
          : `/${parts.path}`
        : "/rest";

      const query = new URLSearchParams();
      if (parts.insecureSkipVerify) {
        query.set("insecureSkipVerify", "true");
      }

      const q = query.toString();
      return `routeros://${user}:${password}@${host}${port}${p}${q ? `?${q}` : ""}`;
    }
    default:
      // Simple types: linux://, darwin://, networkmanager://, etc.
      return `${parts.type}://`;
  }
}

function normalizeBackendUrl(raw: string): string {
  return buildBackendUrl(parseBackendUrl(raw));
}

function maskBackendUrl(raw: string): string {
  if (!raw) return raw;
  return raw.replace(
    /^([a-z][a-z0-9+.-]*:\/\/[^/?#@:\s]+):[^@/?#\s]*@/i,
    "$1:***@"
  );
}

// ─── Form dialog ────────────────────────────────────────────────────────────

function BackendFormDialog({
  backend,
  trigger,
  onClose,
}: {
  backend?: Backend;
  trigger: React.ReactNode;
  onClose?: () => void;
}) {
  const isEditing = !!backend;
  const [open, setOpen] = useState(false);

  // Basic fields
  const [name, setName] = useState(backend?.name ?? "");
  const [description, setDescription] = useState(backend?.description ?? "");
  const [enabled, setEnabled] = useState(backend?.enabled ?? true);

  // URL builder state
  const initialParts = parseBackendUrl(backend?.url ?? "");
  const [backendType, setBackendType] = useState<BackendType>(initialParts.type);
  const [sshHost, setSshHost] = useState(initialParts.host);
  const [sshPort, setSshPort] = useState(initialParts.port);
  const [sshUser, setSshUser] = useState(initialParts.user);
  const [routerPassword, setRouterPassword] = useState(initialParts.password);
  const [routerInsecureSkipVerify, setRouterInsecureSkipVerify] = useState(
    initialParts.insecureSkipVerify
  );
  const [configPath, setConfigPath] = useState(initialParts.path);
  const [useSudo, setUseSudo] = useState(initialParts.sudo);

  const computedUrl = buildBackendUrl({
    type: backendType,
    host: sshHost,
    port: sshPort,
    user: sshUser,
    password: routerPassword,
    path: configPath,
    sudo: useSudo,
    insecureSkipVerify: routerInsecureSkipVerify,
  });

  const typeInfo = getTypeMeta(backendType);

  const { data: availData } = useQuery(AVAILABLE_BACKENDS_QUERY);
  const availableBackends: AvailableBackend[] = availData?.availableBackends ?? [];
  const availableMap = new Map(availableBackends.map((ab) => [ab.type, ab]));

  // Build a merged list: all known types + any extra from the API
  const allKnownTypes = Object.keys(backendTypeMeta);
  const apiOnlyTypes = availableBackends
    .filter((ab) => !allKnownTypes.includes(ab.type))
    .map((ab) => ab.type);
  const mergedTypeKeys = [...allKnownTypes, ...apiOnlyTypes];

  // Determine disabled state per type
  const getTypeDisabled = (typeKey: string): { disabled: boolean; reason: string } => {
    const ab = availableMap.get(typeKey);
    // Not returned by the API at all -- not supported
    if (!ab) return { disabled: true, reason: "Not supported" };
    // Returned but not supported on this platform
    if (!ab.supported) return { disabled: true, reason: "Not supported on this platform" };
    // When editing, allow the current backend's own type even if registered
    const currentType = isEditing ? parseBackendUrl(backend?.url ?? "").type : null;
    if (ab.registered && typeKey !== currentType) {
      return { disabled: true, reason: "Already in use" };
    }
    return { disabled: false, reason: "" };
  };

  const [createBackend, { loading: creating }] = useMutation(
    CREATE_BACKEND_MUTATION,
    { refetchQueries: [{ query: BACKENDS_QUERY }, { query: AVAILABLE_BACKENDS_QUERY }] }
  );
  const [updateBackend, { loading: updating }] = useMutation(
    UPDATE_BACKEND_MUTATION,
    { refetchQueries: [{ query: BACKENDS_QUERY }, { query: AVAILABLE_BACKENDS_QUERY }] }
  );

  const saving = creating || updating;

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    try {
      if (isEditing) {
        const updateInput: Record<string, unknown> = { id: backend.id };
        const originalUrl = normalizeBackendUrl(backend.url);
        const currentUrl = normalizeBackendUrl(computedUrl);

        if (name !== backend.name) updateInput.name = name;
        if (description !== backend.description) updateInput.description = description;
        if (currentUrl !== originalUrl) updateInput.url = computedUrl;
        if (enabled !== backend.enabled) updateInput.enabled = enabled;

        if (Object.keys(updateInput).length === 1) {
          toast.info("No changes to save");
          setOpen(false);
          onClose?.();
          return;
        }

        await updateBackend({
          variables: {
            input: updateInput,
          },
        });
        toast.success("Backend updated");
      } else {
        await createBackend({
          variables: { input: { name, description, url: computedUrl, enabled } },
        });
        toast.success("Backend created");
      }
      setOpen(false);
      onClose?.();
    } catch (err) {
      toast.error(
        err instanceof Error ? err.message : "Failed to save backend"
      );
    }
  };

  const resetForm = (source?: Backend) => {
    const parts = parseBackendUrl(source?.url ?? "");
    setName(source?.name ?? "");
    setDescription(source?.description ?? "");
    setEnabled(source?.enabled ?? true);
    setSshHost(parts.host);
    setSshPort(parts.port);
    setSshUser(parts.user);
    setRouterPassword(parts.password);
    setRouterInsecureSkipVerify(parts.insecureSkipVerify);
    setConfigPath(parts.path);
    setUseSudo(parts.sudo);

    if (source) {
      // Editing: use the backend's current type
      setBackendType(parts.type);
    } else {
      // Creating: pick the first type that is selectable
      const firstAvailable = mergedTypeKeys.find(
        (tk) => !getTypeDisabled(tk).disabled
      );
      setBackendType(firstAvailable ?? "linux");
    }
  };

  const handleOpenChange = (next: boolean) => {
    setOpen(next);
    if (next) {
      resetForm(backend);
    }
  };

  const requiresHost = backendType === "ssh" || backendType === "routeros";
  const isValid =
    name.trim().length > 0 &&
    (!requiresHost || sshHost.trim().length > 0) &&
    (backendType !== "routeros" ||
      (sshUser.trim().length > 0 && routerPassword.trim().length > 0));

  return (
    <Dialog open={open} onOpenChange={handleOpenChange}>
      <DialogTrigger asChild>{trigger}</DialogTrigger>
      <DialogContent className="max-w-lg">
        <form onSubmit={handleSubmit}>
          <DialogHeader>
            <DialogTitle>
              {isEditing ? "Edit Backend" : "New Backend"}
            </DialogTitle>
            <DialogDescription>
              {isEditing
                ? "Update the backend connection settings."
                : "Add a new backend to manage WireGuard servers."}
            </DialogDescription>
          </DialogHeader>

          <div className="flex flex-col gap-4 py-4">
            {/* Name */}
            <div className="flex flex-col gap-1.5">
              <Label htmlFor="backend-name">Name</Label>
              <Input
                id="backend-name"
                value={name}
                onChange={(e) => setName(e.target.value)}
                placeholder="production-server"
                pattern="^[a-zA-Z0-9.\-_]{1,64}$"
                title="1-64 characters: letters, numbers, dots, hyphens, underscores"
                required
              />
            </div>

            {/* Description */}
            <div className="flex flex-col gap-1.5">
              <Label htmlFor="backend-description">Description</Label>
              <Input
                id="backend-description"
                value={description}
                onChange={(e) => setDescription(e.target.value)}
                placeholder="Main production backend"
                maxLength={255}
              />
            </div>

            {/* Backend type selector */}
            <div className="flex flex-col gap-1.5">
              <Label>Backend Type</Label>
              <div className="grid grid-cols-1 gap-1.5">
                {mergedTypeKeys.map((typeKey) => {
                  const meta = getTypeMeta(typeKey);
                  const { disabled, reason } = getTypeDisabled(typeKey);
                  const selected = backendType === typeKey;
                  return (
                    <button
                      key={typeKey}
                      type="button"
                      onClick={() => !disabled && setBackendType(typeKey)}
                      disabled={disabled}
                      className={`flex items-start gap-3 rounded-md border p-3 text-left transition-colors ${
                        disabled
                          ? "cursor-not-allowed border-border bg-muted/40 opacity-50"
                          : selected
                            ? "border-primary bg-primary/5"
                            : "border-border hover:border-muted-foreground/30"
                      }`}
                    >
                      <div
                        className={`mt-0.5 h-4 w-4 shrink-0 rounded-full border-2 ${
                          selected
                            ? "border-primary bg-primary"
                            : "border-muted-foreground/40"
                        }`}
                      >
                        {selected && (
                          <div className="m-auto mt-[3px] h-1.5 w-1.5 rounded-full bg-primary-foreground" />
                        )}
                      </div>
                      <div className="flex-1">
                        <div className="flex items-center gap-2">
                          <span className={`text-sm font-medium ${disabled ? "text-muted-foreground" : "text-foreground"}`}>
                            {meta.label}
                          </span>
                          {disabled && reason && (
                            <Badge variant="secondary" className="text-[10px] px-1.5 py-0">
                              {reason}
                            </Badge>
                          )}
                        </div>
                        <p className="text-xs text-muted-foreground">
                          {meta.description}
                        </p>
                      </div>
                    </button>
                  );
                })}
              </div>
            </div>

            {/* Type-specific fields */}
            {backendType === "ssh" && (
              <div className="flex flex-col gap-3 rounded-md border border-border bg-muted/30 p-3">
                <p className="text-xs font-medium text-muted-foreground">
                  SSH Connection
                </p>
                <div className="grid grid-cols-3 gap-3">
                  <div className="col-span-2 flex flex-col gap-1.5">
                    <Label htmlFor="ssh-host" className="text-xs">
                      Host
                    </Label>
                    <Input
                      id="ssh-host"
                      value={sshHost}
                      onChange={(e) => setSshHost(e.target.value)}
                      placeholder="192.168.1.1 or vpn.example.com"
                      required
                    />
                  </div>
                  <div className="flex flex-col gap-1.5">
                    <Label htmlFor="ssh-port" className="text-xs">
                      Port
                    </Label>
                    <Input
                      id="ssh-port"
                      type="number"
                      min={1}
                      max={65535}
                      value={sshPort}
                      onChange={(e) => setSshPort(e.target.value)}
                      placeholder="22"
                    />
                  </div>
                </div>
                <div className="flex flex-col gap-1.5">
                  <Label htmlFor="ssh-user" className="text-xs">
                    Username
                  </Label>
                  <Input
                    id="ssh-user"
                    value={sshUser}
                    onChange={(e) => setSshUser(e.target.value)}
                    placeholder="root"
                  />
                </div>
                <div className="flex flex-col gap-1.5">
                  <Label htmlFor="ssh-path" className="text-xs">
                    Config Directory
                  </Label>
                  <Input
                    id="ssh-path"
                    value={configPath}
                    onChange={(e) => setConfigPath(e.target.value)}
                    placeholder="/etc/wireguard"
                  />
                </div>
                <label className="flex items-center gap-2 cursor-pointer">
                  <Checkbox
                    checked={useSudo}
                    onCheckedChange={(v) => setUseSudo(v === true)}
                  />
                  <span className="text-sm">Use sudo</span>
                </label>
              </div>
            )}

            {backendType === "exec" && (
              <div className="flex flex-col gap-3 rounded-md border border-border bg-muted/30 p-3">
                <p className="text-xs font-medium text-muted-foreground">
                  Exec Configuration
                </p>
                <div className="flex flex-col gap-1.5">
                  <Label htmlFor="exec-path" className="text-xs">
                    Config Directory
                  </Label>
                  <Input
                    id="exec-path"
                    value={configPath}
                    onChange={(e) => setConfigPath(e.target.value)}
                    placeholder="/etc/wireguard"
                  />
                </div>
                <label className="flex items-center gap-2 cursor-pointer">
                  <Checkbox
                    checked={useSudo}
                    onCheckedChange={(v) => setUseSudo(v === true)}
                  />
                  <span className="text-sm">Use sudo</span>
                </label>
              </div>
            )}

            {backendType === "routeros" && (
              <div className="flex flex-col gap-3 rounded-md border border-border bg-muted/30 p-3">
                <p className="text-xs font-medium text-muted-foreground">
                  RouterOS API
                </p>
                <div className="grid grid-cols-3 gap-3">
                  <div className="col-span-2 flex flex-col gap-1.5">
                    <Label htmlFor="routeros-host" className="text-xs">
                      Host
                    </Label>
                    <Input
                      id="routeros-host"
                      value={sshHost}
                      onChange={(e) => setSshHost(e.target.value)}
                      placeholder="192.168.88.1"
                      required
                    />
                  </div>
                  <div className="flex flex-col gap-1.5">
                    <Label htmlFor="routeros-port" className="text-xs">
                      Port
                    </Label>
                    <Input
                      id="routeros-port"
                      type="number"
                      min={1}
                      max={65535}
                      value={sshPort}
                      onChange={(e) => setSshPort(e.target.value)}
                      placeholder="443"
                    />
                  </div>
                </div>
                <div className="grid grid-cols-2 gap-3">
                  <div className="flex flex-col gap-1.5">
                    <Label htmlFor="routeros-user" className="text-xs">
                      Username
                    </Label>
                    <Input
                      id="routeros-user"
                      value={sshUser}
                      onChange={(e) => setSshUser(e.target.value)}
                      placeholder="admin"
                    />
                  </div>
                  <div className="flex flex-col gap-1.5">
                    <Label htmlFor="routeros-password" className="text-xs">
                      Password
                    </Label>
                    <Input
                      id="routeros-password"
                      type="password"
                      value={routerPassword}
                      onChange={(e) => setRouterPassword(e.target.value)}
                      placeholder="password"
                    />
                  </div>
                </div>
                <div className="flex flex-col gap-1.5">
                  <Label htmlFor="routeros-path" className="text-xs">
                    API Path
                  </Label>
                  <Input
                    id="routeros-path"
                    value={configPath}
                    onChange={(e) => setConfigPath(e.target.value)}
                    placeholder="/rest"
                  />
                </div>
                <label className="flex items-center gap-2 cursor-pointer">
                  <Checkbox
                    checked={routerInsecureSkipVerify}
                    onCheckedChange={(v) => setRouterInsecureSkipVerify(v === true)}
                  />
                  <span className="text-sm">
                    Ignore TLS certificate errors (`insecureSkipVerify=true`)
                  </span>
                </label>
              </div>
            )}

            {!typeInfo.hasFields && (
              <p className="rounded-md border border-border bg-muted/30 p-3 text-xs text-muted-foreground">
                This backend manages WireGuard locally. No additional
                configuration is needed.
              </p>
            )}

            {/* URL preview */}
            <div className="flex flex-col gap-1.5">
              <Label className="text-xs text-muted-foreground">
                Generated URL
              </Label>
              <div className="rounded-md border border-border bg-muted/50 px-3 py-2">
                <code className="text-xs font-mono text-foreground break-all">
                  {computedUrl}
                </code>
              </div>
            </div>

            {/* Enabled toggle */}
            <div className="flex items-center justify-between rounded-md border border-border p-3">
              <div>
                <Label>Enabled</Label>
                <p className="text-xs text-muted-foreground">
                  Whether this backend is active
                </p>
              </div>
              <Switch checked={enabled} onCheckedChange={setEnabled} />
            </div>
          </div>

          <DialogFooter>
            <Button
              type="button"
              variant="outline"
              onClick={() => setOpen(false)}
            >
              Cancel
            </Button>
            <Button type="submit" disabled={saving || !isValid}>
              {saving ? (
                <>
                  <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                  {isEditing ? "Saving..." : "Creating..."}
                </>
              ) : isEditing ? (
                "Save Changes"
              ) : (
                "Create Backend"
              )}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  );
}

export default function BackendsPage() {
  const navigate = useNavigate();
  const [search, setSearch] = useState("");
  const [sortKey, setSortKey] = useState<SortKey>("name");
  const [sortDir, setSortDir] = useState<SortDir>("asc");
  const { data, loading } = useQuery(BACKENDS_QUERY);
  const [deleteBackend] = useMutation(DELETE_BACKEND_MUTATION, {
    refetchQueries: [{ query: BACKENDS_QUERY }, { query: AVAILABLE_BACKENDS_QUERY }],
  });

  const handleSort = (key: SortKey) => {
    if (key === sortKey) {
      setSortDir((d) => (d === "asc" ? "desc" : "asc"));
    } else {
      setSortKey(key);
      setSortDir("asc");
    }
  };

  const handleDelete = async (id: string) => {
    try {
      await deleteBackend({ variables: { input: { id } } });
      toast.success("Backend deleted");
    } catch (err) {
      toast.error(err instanceof Error ? err.message : "Failed to delete");
    }
  };

  const backends: Backend[] = data?.backends ?? [];
  const filtered = search
    ? backends.filter(
        (b) =>
          b.name.toLowerCase().includes(search.toLowerCase()) ||
          b.url.toLowerCase().includes(search.toLowerCase())
      )
    : backends;

  const sorted = useMemo(() => {
    const copy = [...filtered];
    const dir = sortDir === "asc" ? 1 : -1;
    copy.sort((a, b) => {
      switch (sortKey) {
        case "name":
          return dir * a.name.localeCompare(b.name);
        case "url":
          return dir * a.url.localeCompare(b.url);
        case "servers":
          return dir * ((a.servers?.length ?? 0) - (b.servers?.length ?? 0));
        case "status": {
          const aVal = a.enabled ? 1 : 0;
          const bVal = b.enabled ? 1 : 0;
          return dir * (aVal - bVal);
        }
        default:
          return 0;
      }
    });
    return copy;
  }, [filtered, sortKey, sortDir]);

  return (
    <div className="flex flex-col gap-6">
      <div className="flex flex-col gap-4 sm:flex-row sm:items-center sm:justify-between">
        <div>
          <h1 className="text-2xl font-semibold tracking-tight text-foreground">
            Backends
          </h1>
          <p className="mt-1 text-sm text-muted-foreground">
            Manage backend connections for WireGuard server management
          </p>
        </div>
        <BackendFormDialog
          trigger={
            <Button>
              <Plus className="mr-1.5 h-4 w-4" />
              New Backend
            </Button>
          }
        />
      </div>

      {backends.length > 0 && (
        <div className="relative max-w-sm">
          <Search className="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground" />
          <Input
            placeholder="Search backends..."
            value={search}
            onChange={(e) => setSearch(e.target.value)}
            className="pl-9"
          />
        </div>
      )}

      {loading ? (
        <div className="flex items-center justify-center py-16">
          <Loader2 className="h-6 w-6 animate-spin text-muted-foreground" />
        </div>
      ) : sorted.length === 0 ? (
        <div className="flex flex-col items-center justify-center rounded-lg border border-dashed border-border py-16">
          <HardDrive className="mb-3 h-10 w-10 text-muted-foreground/50" />
          <p className="text-sm font-medium text-foreground">
            {backends.length === 0 ? "No backends yet" : "No matching backends"}
          </p>
          <p className="mt-1 text-sm text-muted-foreground">
            {backends.length === 0
              ? "Add a backend to start managing WireGuard servers."
              : "Try a different search term."}
          </p>
        </div>
      ) : (
        <div className="rounded-lg border border-border">
          <Table>
            <TableHeader>
              <TableRow>
                <SortableHead label="Name" sortKey="name" currentKey={sortKey} currentDir={sortDir} onSort={handleSort} />
                <SortableHead label="URL" sortKey="url" currentKey={sortKey} currentDir={sortDir} onSort={handleSort} className="hidden md:table-cell" />
                <SortableHead label="Servers" sortKey="servers" currentKey={sortKey} currentDir={sortDir} onSort={handleSort} />
                <SortableHead label="Status" sortKey="status" currentKey={sortKey} currentDir={sortDir} onSort={handleSort} />
                <TableHead className="w-12">
                  <span className="sr-only">Actions</span>
                </TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {sorted.map((backend) => (
                <TableRow key={backend.id}>
                  <TableCell>
                    <button
                      type="button"
                      onClick={() => navigate(`/servers?backend=${backend.id}`)}
                      className="font-medium text-foreground hover:text-primary"
                    >
                      {backend.name}
                    </button>
                    {backend.description && (
                      <p className="mt-0.5 max-w-xs truncate text-xs text-muted-foreground">
                        {backend.description}
                      </p>
                    )}
                  </TableCell>
                  <TableCell className="hidden md:table-cell">
                    <div className="flex flex-col gap-0.5">
                      <Badge variant="secondary" className="w-fit text-xs">
                        {parseBackendUrl(backend.url).type}
                      </Badge>
                      <span className="font-mono text-xs text-muted-foreground break-all">
                        {maskBackendUrl(backend.url)}
                      </span>
                    </div>
                  </TableCell>
                  <TableCell>
                    <button
                      type="button"
                      onClick={() => navigate(`/servers?backend=${backend.id}`)}
                      className="inline-flex items-center gap-1.5 text-sm text-muted-foreground hover:text-primary"
                    >
                      {backend.servers?.length ?? 0}
                      <ExternalLink className="h-3 w-3" />
                    </button>
                  </TableCell>
                  <TableCell>
                    <div className="flex items-center gap-2">
                      {backend.enabled ? (
                        <Badge
                          variant="outline"
                          className="border-success/30 bg-success/10 text-success"
                        >
                          Enabled
                        </Badge>
                      ) : (
                        <Badge
                          variant="outline"
                          className="border-muted-foreground/30 text-muted-foreground"
                        >
                          Disabled
                        </Badge>
                      )}
                      {!backend.supported && (
                        <Badge variant="secondary" className="text-xs">
                          Unsupported
                        </Badge>
                      )}
                    </div>
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
                        <DropdownMenuItem
                          onClick={() => navigate(`/servers?backend=${backend.id}`)}
                        >
                          <ExternalLink className="mr-2 h-4 w-4" />
                          View Servers
                        </DropdownMenuItem>
                        <BackendFormDialog
                          backend={backend}
                          trigger={
                            <DropdownMenuItem onSelect={(e) => e.preventDefault()}>
                              <Pencil className="mr-2 h-4 w-4" />
                              Edit
                            </DropdownMenuItem>
                          }
                        />
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
                          title="Delete Backend"
                          description={`Are you sure you want to delete "${backend.name}"? This will also remove all servers and peers on this backend. This action cannot be undone.`}
                          confirmLabel="Delete Backend"
                          onConfirm={() => handleDelete(backend.id)}
                        />
                      </DropdownMenuContent>
                    </DropdownMenu>
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
