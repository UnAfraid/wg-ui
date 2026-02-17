"use client";

import { useState } from "react";
import { Link, useNavigate } from "react-router-dom";
import { useMutation, useQuery } from "@apollo/client";
import { useForm } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import { z } from "zod";
import {
  ArrowLeft,
  ChevronDown,
  Server,
  MonitorSmartphone,
  Settings2,
  Loader2,
  KeyRound,
} from "lucide-react";
import { toast } from "sonner";

import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Textarea } from "@/components/ui/textarea";
import { Switch } from "@/components/ui/switch";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import {
  Form,
  FormControl,
  FormDescription,
  FormField,
  FormItem,
  FormLabel,
  FormMessage,
} from "@/components/ui/form";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { HooksEditor, type ServerHookValue } from "@/components/hooks-editor";
import {
  CREATE_SERVER_MUTATION,
  CREATE_PEER_MUTATION,
  GENERATE_WIREGUARD_KEY_MUTATION,
} from "@/lib/graphql/mutations";
import { SERVERS_QUERY, BACKENDS_QUERY } from "@/lib/graphql/queries";
import type { Backend } from "@/lib/graphql/types";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";

type Mode = "centralized" | "client" | "advanced";

const modeConfig = {
  centralized: {
    label: "Centralized Server",
    description: "Create a WireGuard server and register peers that connect to it",
    icon: Server,
  },
  client: {
    label: "Client (Reverse)",
    description: "Connect to a pre-existing WireGuard server as a client",
    icon: MonitorSmartphone,
  },
  advanced: {
    label: "Advanced",
    description: "Full control over all interface configuration options",
    icon: Settings2,
  },
};

const centralizedSchema = z.object({
  name: z.string().min(1, "Name is required"),
  description: z.string().optional(),
  address: z.string().min(1, "Address is required"),
  listenPort: z.coerce.number().int().min(1).max(65535),
  dns: z.string().optional(),
  privateKey: z.string().optional(),
  enabled: z.boolean(),
});

const clientSchema = z.object({
  name: z.string().min(1, "Name is required"),
  description: z.string().optional(),
  address: z.string().min(1, "Local tunnel address is required"),
  dns: z.string().optional(),
  privateKey: z.string().optional(),
  remoteEndpoint: z.string().min(1, "Remote server endpoint is required"),
  remotePublicKey: z.string().min(1, "Remote server public key is required"),
  remotePresharedKey: z.string().optional(),
  allowedIPs: z.string().optional(),
  enabled: z.boolean(),
});

const advancedSchema = z.object({
  name: z.string().min(1, "Name is required"),
  description: z.string().optional(),
  address: z.string().min(1, "Address is required"),
  listenPort: z.coerce.number().int().min(1).max(65535).optional().or(z.literal("")),
  dns: z.string().optional(),
  mtu: z.coerce.number().int().min(1280).max(9000).optional().or(z.literal("")),
  firewallMark: z.coerce.number().int().optional().or(z.literal("")),
  privateKey: z.string().optional(),
  enabled: z.boolean(),
});

export default function NewServerPage() {
  const [mode, setMode] = useState<Mode>("centralized");
  const [backendId, setBackendId] = useState<string>("");
  const navigate = useNavigate();
  const { data: backendsData } = useQuery(BACKENDS_QUERY);
  const backends: Backend[] = backendsData?.backends ?? [];

  // Auto-select first backend
  if (backends.length > 0 && !backendId) {
    setBackendId(backends[0].id);
  }

  return (
    <div className="flex flex-col gap-6">
      <div className="flex items-center gap-3">
        <Button variant="ghost" size="icon" className="h-8 w-8" asChild>
          <Link to="/servers">
            <ArrowLeft className="h-4 w-4" />
          </Link>
        </Button>
        <div>
          <h1 className="text-2xl font-semibold tracking-tight text-foreground">
            New Server
          </h1>
          <p className="mt-0.5 text-sm text-muted-foreground">
            Create a new WireGuard interface
          </p>
        </div>
      </div>

      <div className="flex flex-wrap items-center gap-4">
        <div className="flex items-center gap-3">
          <span className="text-sm text-muted-foreground">Backend:</span>
          <Select value={backendId} onValueChange={setBackendId}>
            <SelectTrigger className="w-52">
              <SelectValue placeholder="Select a backend" />
            </SelectTrigger>
            <SelectContent>
              {backends.map((b) => (
                <SelectItem key={b.id} value={b.id}>
                  {b.name}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
        </div>

        <div className="flex items-center gap-3">
          <span className="text-sm text-muted-foreground">Mode:</span>
          <DropdownMenu>
            <DropdownMenuTrigger asChild>
              <Button variant="outline" className="gap-2">
                {(() => {
                  const Icon = modeConfig[mode].icon;
                  return <Icon className="h-4 w-4" />;
                })()}
                {modeConfig[mode].label}
                <ChevronDown className="h-3 w-3 text-muted-foreground" />
              </Button>
            </DropdownMenuTrigger>
            <DropdownMenuContent align="start" className="w-72">
              {(Object.keys(modeConfig) as Mode[]).map((key) => {
                const Icon = modeConfig[key].icon;
                return (
                  <DropdownMenuItem
                    key={key}
                    onClick={() => setMode(key)}
                    className="flex flex-col items-start gap-0.5 py-2"
                  >
                    <div className="flex items-center gap-2">
                      <Icon className="h-4 w-4" />
                      <span className="font-medium">{modeConfig[key].label}</span>
                      {mode === key && (
                        <Badge variant="secondary" className="ml-1 text-[10px]">
                          Selected
                        </Badge>
                      )}
                    </div>
                    <span className="ml-6 text-xs text-muted-foreground">
                      {modeConfig[key].description}
                    </span>
                  </DropdownMenuItem>
                );
              })}
            </DropdownMenuContent>
          </DropdownMenu>
        </div>
      </div>

      {mode === "centralized" && <CentralizedForm backendId={backendId} />}
      {mode === "client" && <ClientForm backendId={backendId} />}
      {mode === "advanced" && <AdvancedForm backendId={backendId} />}
    </div>
  );
}

function CentralizedForm({ backendId }: { backendId: string }) {
  const navigate = useNavigate();

  const [generateKey, { loading: generatingKey }] = useMutation(
    GENERATE_WIREGUARD_KEY_MUTATION
  );
  const [createServer, { loading }] = useMutation(CREATE_SERVER_MUTATION, {
    refetchQueries: [{ query: SERVERS_QUERY }],
  });

  const form = useForm<z.infer<typeof centralizedSchema>>({
    resolver: zodResolver(centralizedSchema),
    defaultValues: {
      name: "",
      description: "",
      address: "10.0.0.1/24",
      listenPort: 51820,
      dns: "1.1.1.1",
      privateKey: "",
      enabled: true,
    },
  });

  const handleGenerateKey = async () => {
    try {
      const { data } = await generateKey({ variables: { input: {} } });
      if (data?.generateWireguardKey) {
        form.setValue("privateKey", data.generateWireguardKey.privateKey);
        toast.success("Key pair generated");
      }
    } catch (err) {
      toast.error(err instanceof Error ? err.message : "Failed to generate key");
    }
  };

  const onSubmit = async (values: z.infer<typeof centralizedSchema>) => {
    try {
      const dnsArray = values.dns
        ? values.dns.split(",").map((d) => d.trim()).filter(Boolean)
        : undefined;

      const { data } = await createServer({
        variables: {
          input: {
            backendId,
            name: values.name,
            description: values.description || "",
            address: values.address,
            listenPort: values.listenPort,
            dns: dnsArray,
            privateKey: values.privateKey || undefined,
            enabled: values.enabled,
          },
        },
      });

      toast.success("Server created successfully");
      const newId = data?.createServer?.server?.id;
      navigate(newId ? `/servers/${newId}` : "/servers");
    } catch (err) {
      toast.error(err instanceof Error ? err.message : "Failed to create server");
    }
  };

  return (
    <Form {...form}>
      <form onSubmit={form.handleSubmit(onSubmit)} className="flex flex-col gap-6">
        <Card>
          <CardHeader>
            <CardTitle className="text-base">Server Configuration</CardTitle>
          </CardHeader>
          <CardContent className="flex flex-col gap-4">
            <FormField
              control={form.control}
              name="name"
              render={({ field }) => (
                <FormItem>
                  <FormLabel>Name</FormLabel>
                  <FormControl>
                    <Input placeholder="wg0" {...field} />
                  </FormControl>
                  <FormMessage />
                </FormItem>
              )}
            />
            <FormField
              control={form.control}
              name="description"
              render={({ field }) => (
                <FormItem>
                  <FormLabel>Description</FormLabel>
                  <FormControl>
                    <Textarea placeholder="My VPN server..." rows={2} {...field} />
                  </FormControl>
                  <FormMessage />
                </FormItem>
              )}
            />
            <FormField
              control={form.control}
              name="address"
              render={({ field }) => (
                <FormItem>
                  <FormLabel>Address</FormLabel>
                  <FormControl>
                    <Input placeholder="10.0.0.1/24" {...field} />
                  </FormControl>
                  <FormDescription>
                    The server IP and subnet (e.g. 10.0.0.1/24)
                  </FormDescription>
                  <FormMessage />
                </FormItem>
              )}
            />
            <div className="grid grid-cols-1 gap-4 md:grid-cols-2">
              <FormField
                control={form.control}
                name="listenPort"
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>Listen Port</FormLabel>
                    <FormControl>
                      <Input type="number" placeholder="51820" {...field} />
                    </FormControl>
                    <FormMessage />
                  </FormItem>
                )}
              />
              <FormField
                control={form.control}
                name="dns"
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>DNS Servers</FormLabel>
                    <FormControl>
                      <Input placeholder="1.1.1.1, 8.8.8.8" {...field} />
                    </FormControl>
                    <FormMessage />
                  </FormItem>
                )}
              />
            </div>
            <FormField
              control={form.control}
              name="enabled"
              render={({ field }) => (
                <FormItem className="flex items-center justify-between rounded-md border border-border p-3">
                  <div>
                    <FormLabel>Enabled</FormLabel>
                    <FormDescription>Start the server after creation</FormDescription>
                  </div>
                  <FormControl>
                    <Switch checked={field.value} onCheckedChange={field.onChange} />
                  </FormControl>
                </FormItem>
              )}
            />
          </CardContent>
        </Card>

        <Card>
          <CardHeader>
            <div className="flex items-center justify-between">
              <CardTitle className="text-base">Key Pair</CardTitle>
              <Button
                type="button"
                variant="outline"
                size="sm"
                onClick={handleGenerateKey}
                disabled={generatingKey}
              >
                {generatingKey ? (
                  <Loader2 className="mr-1.5 h-3 w-3 animate-spin" />
                ) : (
                  <KeyRound className="mr-1.5 h-3 w-3" />
                )}
                Generate
              </Button>
            </div>
          </CardHeader>
          <CardContent>
            <FormField
              control={form.control}
              name="privateKey"
              render={({ field }) => (
                <FormItem>
                  <FormLabel>Private Key</FormLabel>
                  <FormControl>
                    <Input
                      placeholder="Auto-generated if empty"
                      className="font-mono text-sm"
                      {...field}
                    />
                  </FormControl>
                  <FormDescription>
                    Click Generate to create a new key pair, or paste an existing private key. Left empty, the server will auto-generate one.
                  </FormDescription>
                  <FormMessage />
                </FormItem>
              )}
            />
          </CardContent>
        </Card>

        <div className="flex items-center gap-3">
          <Button type="submit" disabled={loading}>
            {loading ? (
              <>
                <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                Creating...
              </>
            ) : (
              "Create Server"
            )}
          </Button>
          <Button type="button" variant="outline" onClick={() => navigate(-1)}>
            Cancel
          </Button>
        </div>
      </form>
    </Form>
  );
}

function ClientForm({ backendId }: { backendId: string }) {
  const navigate = useNavigate();

  const [generateKey, { loading: generatingKey }] = useMutation(
    GENERATE_WIREGUARD_KEY_MUTATION
  );
  const [createServer, { loading: creatingServer }] = useMutation(
    CREATE_SERVER_MUTATION,
    { refetchQueries: [{ query: SERVERS_QUERY }] }
  );
  const [createPeer, { loading: creatingPeer }] = useMutation(
    CREATE_PEER_MUTATION
  );

  const form = useForm<z.infer<typeof clientSchema>>({
    resolver: zodResolver(clientSchema),
    defaultValues: {
      name: "",
      description: "",
      address: "10.0.0.2/32",
      dns: "1.1.1.1",
      privateKey: "",
      remoteEndpoint: "",
      remotePublicKey: "",
      remotePresharedKey: "",
      allowedIPs: "0.0.0.0/0, ::/0",
      enabled: true,
    },
  });

  const saving = creatingServer || creatingPeer;

  const handleGenerateKey = async () => {
    try {
      const { data } = await generateKey({ variables: { input: {} } });
      if (data?.generateWireguardKey) {
        form.setValue("privateKey", data.generateWireguardKey.privateKey);
        toast.success("Key pair generated");
      }
    } catch (err) {
      toast.error(err instanceof Error ? err.message : "Failed to generate key");
    }
  };

  const onSubmit = async (values: z.infer<typeof clientSchema>) => {
    try {
      const dnsArray = values.dns
        ? values.dns.split(",").map((d) => d.trim()).filter(Boolean)
        : undefined;

      // Step 1: Create the local interface (server)
      const { data: serverData } = await createServer({
        variables: {
          input: {
            backendId,
            name: values.name,
            description: values.description || "Client interface",
            address: values.address,
            dns: dnsArray,
            privateKey: values.privateKey || undefined,
            enabled: values.enabled,
          },
        },
      });

      const serverId = serverData?.createServer?.server?.id;
      if (!serverId) throw new Error("Failed to create interface");

      // Step 2: Add the remote server as a peer
      const allowedIPs = values.allowedIPs
        ? values.allowedIPs.split(",").map((ip) => ip.trim()).filter(Boolean)
        : ["0.0.0.0/0", "::/0"];

      await createPeer({
        variables: {
          input: {
            serverId,
            name: "Remote Server",
            publicKey: values.remotePublicKey,
            endpoint: values.remoteEndpoint,
            allowedIPs,
            presharedKey: values.remotePresharedKey || undefined,
            persistentKeepalive: 25,
          },
        },
      });

      toast.success("Client interface created with remote peer");
      navigate(`/servers/${serverId}`);
    } catch (err) {
      toast.error(err instanceof Error ? err.message : "Failed to create client");
    }
  };

  return (
    <Form {...form}>
      <form onSubmit={form.handleSubmit(onSubmit)} className="flex flex-col gap-6">
        <Card>
          <CardHeader>
            <CardTitle className="text-base">Local Interface</CardTitle>
          </CardHeader>
          <CardContent className="flex flex-col gap-4">
            <FormField
              control={form.control}
              name="name"
              render={({ field }) => (
                <FormItem>
                  <FormLabel>Interface Name</FormLabel>
                  <FormControl>
                    <Input placeholder="wg-client" {...field} />
                  </FormControl>
                  <FormMessage />
                </FormItem>
              )}
            />
            <FormField
              control={form.control}
              name="description"
              render={({ field }) => (
                <FormItem>
                  <FormLabel>Description</FormLabel>
                  <FormControl>
                    <Textarea placeholder="VPN client to office..." rows={2} {...field} />
                  </FormControl>
                  <FormMessage />
                </FormItem>
              )}
            />
            <FormField
              control={form.control}
              name="address"
              render={({ field }) => (
                <FormItem>
                  <FormLabel>Local Tunnel Address</FormLabel>
                  <FormControl>
                    <Input placeholder="10.0.0.2/32" {...field} />
                  </FormControl>
                  <FormDescription>
                    Your assigned IP address in the remote network
                  </FormDescription>
                  <FormMessage />
                </FormItem>
              )}
            />
            <FormField
              control={form.control}
              name="dns"
              render={({ field }) => (
                <FormItem>
                  <FormLabel>DNS Servers</FormLabel>
                  <FormControl>
                    <Input placeholder="1.1.1.1, 8.8.8.8" {...field} />
                  </FormControl>
                  <FormMessage />
                </FormItem>
              )}
            />
            <FormField
              control={form.control}
              name="enabled"
              render={({ field }) => (
                <FormItem className="flex items-center justify-between rounded-md border border-border p-3">
                  <div>
                    <FormLabel>Enabled</FormLabel>
                    <FormDescription>Start the interface after creation</FormDescription>
                  </div>
                  <FormControl>
                    <Switch checked={field.value} onCheckedChange={field.onChange} />
                  </FormControl>
                </FormItem>
              )}
            />
          </CardContent>
        </Card>

        <Card>
          <CardHeader>
            <div className="flex items-center justify-between">
              <CardTitle className="text-base">Key Pair</CardTitle>
              <Button
                type="button"
                variant="outline"
                size="sm"
                onClick={handleGenerateKey}
                disabled={generatingKey}
              >
                {generatingKey ? (
                  <Loader2 className="mr-1.5 h-3 w-3 animate-spin" />
                ) : (
                  <KeyRound className="mr-1.5 h-3 w-3" />
                )}
                Generate
              </Button>
            </div>
          </CardHeader>
          <CardContent>
            <FormField
              control={form.control}
              name="privateKey"
              render={({ field }) => (
                <FormItem>
                  <FormLabel>Private Key</FormLabel>
                  <FormControl>
                    <Input
                      placeholder="Auto-generated if empty"
                      className="font-mono text-sm"
                      {...field}
                    />
                  </FormControl>
                  <FormDescription>
                    Click Generate to create a new key pair, or paste an existing private key. Left empty, the server will auto-generate one.
                  </FormDescription>
                  <FormMessage />
                </FormItem>
              )}
            />
          </CardContent>
        </Card>

        <Card>
          <CardHeader>
            <CardTitle className="text-base">Remote Server</CardTitle>
          </CardHeader>
          <CardContent className="flex flex-col gap-4">
            <FormField
              control={form.control}
              name="remoteEndpoint"
              render={({ field }) => (
                <FormItem>
                  <FormLabel>Endpoint</FormLabel>
                  <FormControl>
                    <Input placeholder="vpn.example.com:51820" {...field} />
                  </FormControl>
                  <FormDescription>
                    {"The remote server's address and port"}
                  </FormDescription>
                  <FormMessage />
                </FormItem>
              )}
            />
            <FormField
              control={form.control}
              name="remotePublicKey"
              render={({ field }) => (
                <FormItem>
                  <FormLabel>Public Key</FormLabel>
                  <FormControl>
                    <Input
                      placeholder="Remote server's public key"
                      className="font-mono text-sm"
                      {...field}
                    />
                  </FormControl>
                  <FormMessage />
                </FormItem>
              )}
            />
            <FormField
              control={form.control}
              name="remotePresharedKey"
              render={({ field }) => (
                <FormItem>
                  <FormLabel>Preshared Key (Optional)</FormLabel>
                  <FormControl>
                    <Input
                      placeholder="Optional preshared key"
                      className="font-mono text-sm"
                      {...field}
                    />
                  </FormControl>
                  <FormMessage />
                </FormItem>
              )}
            />
            <FormField
              control={form.control}
              name="allowedIPs"
              render={({ field }) => (
                <FormItem>
                  <FormLabel>Allowed IPs</FormLabel>
                  <FormControl>
                    <Input placeholder="0.0.0.0/0, ::/0" {...field} />
                  </FormControl>
                  <FormDescription>
                    Comma-separated. Use 0.0.0.0/0, ::/0 to route all traffic.
                  </FormDescription>
                  <FormMessage />
                </FormItem>
              )}
            />
          </CardContent>
        </Card>

        <div className="flex items-center gap-3">
          <Button type="submit" disabled={saving}>
            {saving ? (
              <>
                <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                Creating...
              </>
            ) : (
              "Create Client Interface"
            )}
          </Button>
          <Button type="button" variant="outline" onClick={() => navigate(-1)}>
            Cancel
          </Button>
        </div>
      </form>
    </Form>
  );
}

function AdvancedForm({ backendId }: { backendId: string }) {
  const navigate = useNavigate();
  const [hooks, setHooks] = useState<ServerHookValue[]>([]);

  const [generateKey, { loading: generatingKey }] = useMutation(
    GENERATE_WIREGUARD_KEY_MUTATION
  );
  const [createServer, { loading }] = useMutation(CREATE_SERVER_MUTATION, {
    refetchQueries: [{ query: SERVERS_QUERY }],
  });

  const form = useForm<z.infer<typeof advancedSchema>>({
    resolver: zodResolver(advancedSchema),
    defaultValues: {
      name: "",
      description: "",
      address: "",
      listenPort: "" as unknown as undefined,
      dns: "",
      mtu: "" as unknown as undefined,
      firewallMark: "" as unknown as undefined,
      privateKey: "",
      enabled: true,
    },
  });

  const handleGenerateKey = async () => {
    try {
      const { data } = await generateKey({ variables: { input: {} } });
      if (data?.generateWireguardKey) {
        form.setValue("privateKey", data.generateWireguardKey.privateKey);
        toast.success("Key pair generated");
      }
    } catch (err) {
      toast.error(err instanceof Error ? err.message : "Failed to generate key");
    }
  };

  const onSubmit = async (values: z.infer<typeof advancedSchema>) => {
    try {
      const dnsArray = values.dns
        ? values.dns.split(",").map((d) => d.trim()).filter(Boolean)
        : undefined;

      const { data } = await createServer({
        variables: {
          input: {
            backendId,
            name: values.name,
            description: values.description || "",
            address: values.address,
            listenPort: values.listenPort || undefined,
            dns: dnsArray,
            mtu: values.mtu || undefined,
            firewallMark: values.firewallMark || undefined,
            privateKey: values.privateKey || undefined,
            enabled: values.enabled,
            hooks: hooks.length > 0 ? hooks : undefined,
          },
        },
      });

      toast.success("Server created");
      const newId = data?.createServer?.server?.id;
      navigate(newId ? `/servers/${newId}` : "/servers");
    } catch (err) {
      toast.error(err instanceof Error ? err.message : "Failed to create server");
    }
  };

  return (
    <Form {...form}>
      <form onSubmit={form.handleSubmit(onSubmit)} className="flex flex-col gap-6">
        <Card>
          <CardHeader>
            <CardTitle className="text-base">General</CardTitle>
          </CardHeader>
          <CardContent className="flex flex-col gap-4">
            <FormField
              control={form.control}
              name="name"
              render={({ field }) => (
                <FormItem>
                  <FormLabel>Name</FormLabel>
                  <FormControl>
                    <Input placeholder="wg0" {...field} />
                  </FormControl>
                  <FormMessage />
                </FormItem>
              )}
            />
            <FormField
              control={form.control}
              name="description"
              render={({ field }) => (
                <FormItem>
                  <FormLabel>Description</FormLabel>
                  <FormControl>
                    <Textarea placeholder="Optional description..." rows={2} {...field} />
                  </FormControl>
                  <FormMessage />
                </FormItem>
              )}
            />
            <FormField
              control={form.control}
              name="address"
              render={({ field }) => (
                <FormItem>
                  <FormLabel>Address</FormLabel>
                  <FormControl>
                    <Input placeholder="10.0.0.1/24" {...field} />
                  </FormControl>
                  <FormDescription>IP address and CIDR for this interface</FormDescription>
                  <FormMessage />
                </FormItem>
              )}
            />
            <div className="grid grid-cols-1 gap-4 md:grid-cols-2">
              <FormField
                control={form.control}
                name="listenPort"
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>Listen Port</FormLabel>
                    <FormControl>
                      <Input type="number" placeholder="51820" {...field} value={field.value ?? ""} />
                    </FormControl>
                    <FormMessage />
                  </FormItem>
                )}
              />
              <FormField
                control={form.control}
                name="dns"
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>DNS Servers</FormLabel>
                    <FormControl>
                      <Input placeholder="1.1.1.1, 8.8.8.8" {...field} />
                    </FormControl>
                    <FormDescription>Comma-separated</FormDescription>
                    <FormMessage />
                  </FormItem>
                )}
              />
            </div>
            <div className="grid grid-cols-1 gap-4 md:grid-cols-2">
              <FormField
                control={form.control}
                name="mtu"
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>MTU</FormLabel>
                    <FormControl>
                      <Input type="number" placeholder="1420" {...field} value={field.value ?? ""} />
                    </FormControl>
                    <FormMessage />
                  </FormItem>
                )}
              />
              <FormField
                control={form.control}
                name="firewallMark"
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>Firewall Mark</FormLabel>
                    <FormControl>
                      <Input type="number" placeholder="0" {...field} value={field.value ?? ""} />
                    </FormControl>
                    <FormMessage />
                  </FormItem>
                )}
              />
            </div>
            <FormField
              control={form.control}
              name="enabled"
              render={({ field }) => (
                <FormItem className="flex items-center justify-between rounded-md border border-border p-3">
                  <div>
                    <FormLabel>Enabled</FormLabel>
                    <FormDescription>Whether this server should be available</FormDescription>
                  </div>
                  <FormControl>
                    <Switch checked={field.value} onCheckedChange={field.onChange} />
                  </FormControl>
                </FormItem>
              )}
            />
          </CardContent>
        </Card>

        <Card>
          <CardHeader>
            <div className="flex items-center justify-between">
              <CardTitle className="text-base">Key Pair</CardTitle>
              <Button
                type="button"
                variant="outline"
                size="sm"
                onClick={handleGenerateKey}
                disabled={generatingKey}
              >
                {generatingKey ? (
                  <Loader2 className="mr-1.5 h-3 w-3 animate-spin" />
                ) : (
                  <KeyRound className="mr-1.5 h-3 w-3" />
                )}
                Generate
              </Button>
            </div>
          </CardHeader>
          <CardContent>
            <FormField
              control={form.control}
              name="privateKey"
              render={({ field }) => (
                <FormItem>
                  <FormLabel>Private Key</FormLabel>
                  <FormControl>
                    <Input
                      placeholder="Auto-generated if empty"
                      className="font-mono text-sm"
                      {...field}
                    />
                  </FormControl>
                  <FormDescription>
                    Click Generate to create a new key pair, or paste an existing private key.
                  </FormDescription>
                  <FormMessage />
                </FormItem>
              )}
            />
          </CardContent>
        </Card>

        <Card>
          <CardContent className="pt-6">
            <HooksEditor type="server" value={hooks} onChange={setHooks} />
          </CardContent>
        </Card>

        <div className="flex items-center gap-3">
          <Button type="submit" disabled={loading}>
            {loading ? (
              <>
                <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                Creating...
              </>
            ) : (
              "Create Server"
            )}
          </Button>
          <Button type="button" variant="outline" onClick={() => navigate(-1)}>
            Cancel
          </Button>
        </div>
      </form>
    </Form>
  );
}
