"use client";

import { useState } from "react";
import { useForm } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import { z } from "zod";
import { useMutation } from "@apollo/client";
import { useNavigate } from "react-router-dom";
import { Loader2, KeyRound, X } from "lucide-react";
import { toast } from "sonner";

import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Textarea } from "@/components/ui/textarea";
import { Switch } from "@/components/ui/switch";
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
import { Separator } from "@/components/ui/separator";
import { HooksEditor, type ServerHookValue } from "@/components/hooks-editor";
import {
  CREATE_SERVER_MUTATION,
  UPDATE_SERVER_MUTATION,
  GENERATE_WIREGUARD_KEY_MUTATION,
} from "@/lib/graphql/mutations";
import { SERVERS_QUERY, SERVER_QUERY } from "@/lib/graphql/queries";
import type { Server } from "@/lib/graphql/types";

const serverSchema = z.object({
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

type ServerFormValues = z.infer<typeof serverSchema>;

interface ServerFormProps {
  server?: Server;
  showAdvanced?: boolean;
}

function normalizeDnsList(value: string): string[] {
  if (!value.trim()) {
    return [];
  }
  return value
    .split(",")
    .map((d) => d.trim())
    .filter(Boolean);
}

function areStringArraysEqual(a: string[], b: string[]): boolean {
  if (a.length !== b.length) return false;
  return a.every((value, index) => value === b[index]);
}

function normalizeServerHooks(hooks?: Server["hooks"] | null): ServerHookValue[] {
  return (
    hooks?.map((h) => ({
      command: h.command,
      runOnPreUp: h.runOnPreUp,
      runOnPostUp: h.runOnPostUp,
      runOnPreDown: h.runOnPreDown,
      runOnPostDown: h.runOnPostDown,
    })) ?? []
  );
}

export function ServerForm({ server, showAdvanced = false }: ServerFormProps) {
  const navigate = useNavigate();
  const isEditing = !!server;
  const [hooks, setHooks] = useState<ServerHookValue[]>(
    normalizeServerHooks(server?.hooks)
  );
  const [advancedOpen, setAdvancedOpen] = useState(showAdvanced);

  const [generateKey, { loading: generatingKey }] = useMutation(
    GENERATE_WIREGUARD_KEY_MUTATION
  );

  const [createServer, { loading: creating }] = useMutation(
    CREATE_SERVER_MUTATION,
    {
      refetchQueries: [{ query: SERVERS_QUERY }],
    }
  );

  const [updateServer, { loading: updating }] = useMutation(
    UPDATE_SERVER_MUTATION,
    {
      refetchQueries: [
        { query: SERVERS_QUERY },
        ...(server ? [{ query: SERVER_QUERY, variables: { id: server.id } }] : []),
      ],
    }
  );

  const form = useForm<ServerFormValues>({
    resolver: zodResolver(serverSchema),
    defaultValues: {
      name: server?.name ?? "",
      description: server?.description ?? "",
      address: server?.address ?? "",
      listenPort: server?.listenPort ?? ("" as unknown as undefined),
      dns: server?.dns?.join(", ") ?? "",
      mtu: server?.mtu ?? ("" as unknown as undefined),
      firewallMark: server?.firewallMark ?? ("" as unknown as undefined),
      privateKey: "",
      enabled: server?.enabled ?? true,
    },
  });

  const handleGenerateKey = async () => {
    try {
      const { data } = await generateKey({ variables: { input: {} } });
      if (data?.generateWireguardKey) {
        form.setValue("privateKey", data.generateWireguardKey.privateKey);
        toast.success("Key pair generated. Private key has been set.");
      }
    } catch (err) {
      toast.error(
        err instanceof Error ? err.message : "Failed to generate key"
      );
    }
  };

  const onSubmit = async (values: ServerFormValues) => {
    try {
      const dnsArray = normalizeDnsList(values.dns || "");

      if (isEditing) {
        const updateInput: Record<string, unknown> = { id: server.id };
        const description = values.description || "";
        const listenPort = typeof values.listenPort === "number" ? values.listenPort : null;
        const firewallMark =
          typeof values.firewallMark === "number" ? values.firewallMark : null;
        const mtu = typeof values.mtu === "number" ? values.mtu : undefined;
        const currentHooks = hooks.length > 0 ? hooks : [];
        const originalHooks = normalizeServerHooks(server.hooks);
        const originalDns = server.dns ?? [];

        if (description !== (server.description || "")) {
          updateInput.description = description;
        }
        if (values.address !== server.address) {
          updateInput.address = values.address;
        }
        if (listenPort !== (server.listenPort ?? null)) {
          updateInput.listenPort = listenPort;
        }
        if (!areStringArraysEqual(dnsArray, originalDns)) {
          updateInput.dns = dnsArray;
        }
        if (mtu !== undefined && mtu !== server.mtu) {
          updateInput.mtu = mtu;
        }
        if (firewallMark !== (server.firewallMark ?? null)) {
          updateInput.firewallMark = firewallMark;
        }
        if (values.privateKey?.trim()) {
          updateInput.privateKey = values.privateKey.trim();
        }
        if (values.enabled !== server.enabled) {
          updateInput.enabled = values.enabled;
        }
        if (JSON.stringify(currentHooks) !== JSON.stringify(originalHooks)) {
          updateInput.hooks = currentHooks;
        }

        if (Object.keys(updateInput).length === 1) {
          toast.info("No changes to save");
          navigate(`/servers/${server.id}`);
          return;
        }

        await updateServer({
          variables: { input: updateInput },
        });
        toast.success("Server updated");
        navigate(`/servers/${server.id}`);
      } else {
        const { data } = await createServer({
          variables: {
            input: {
              name: values.name,
              description: values.description || "",
              address: values.address,
              listenPort: values.listenPort || undefined,
              dns: dnsArray.length > 0 ? dnsArray : undefined,
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
      }
    } catch (err) {
      toast.error(
        err instanceof Error ? err.message : "Failed to save server"
      );
    }
  };

  const saving = creating || updating;

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
                    <Input placeholder="wg0" {...field} disabled={isEditing} />
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
                    <Textarea
                      placeholder="Optional description..."
                      rows={2}
                      {...field}
                    />
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
                    The IP address and CIDR for this interface (e.g.
                    10.0.0.1/24)
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
                      <Input
                        type="number"
                        placeholder="51820"
                        {...field}
                        value={field.value ?? ""}
                      />
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
                      <Input
                        placeholder="1.1.1.1, 8.8.8.8"
                        {...field}
                      />
                    </FormControl>
                    <FormDescription>Comma-separated</FormDescription>
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
                    <FormDescription>
                      Whether this server should be available for starting
                    </FormDescription>
                  </div>
                  <FormControl>
                    <Switch
                      checked={field.value}
                      onCheckedChange={field.onChange}
                    />
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
                      placeholder={
                        isEditing
                          ? "Leave empty to keep current key"
                          : "Auto-generated if empty"
                      }
                      className="font-mono text-sm"
                      {...field}
                    />
                  </FormControl>
                  <FormDescription>
                    {isEditing
                      ? "Leave empty to keep the current private key. The public key is derived from this."
                      : "Click Generate to create a new key pair, or paste an existing private key."}
                  </FormDescription>
                  <FormMessage />
                </FormItem>
              )}
            />
          </CardContent>
        </Card>

        <div>
          <Button
            type="button"
            variant="ghost"
            className="text-sm text-muted-foreground"
            onClick={() => setAdvancedOpen(!advancedOpen)}
          >
            {advancedOpen ? "Hide" : "Show"} advanced settings
          </Button>
        </div>

        {advancedOpen && (
          <>
            <Card>
              <CardHeader>
                <CardTitle className="text-base">Advanced</CardTitle>
              </CardHeader>
              <CardContent className="flex flex-col gap-4">
                <div className="grid grid-cols-1 gap-4 md:grid-cols-2">
                  <FormField
                    control={form.control}
                    name="mtu"
                    render={({ field }) => (
                      <FormItem>
                        <FormLabel>MTU</FormLabel>
                        <FormControl>
                          <Input
                            type="number"
                            placeholder="1420"
                            {...field}
                            value={field.value ?? ""}
                          />
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
                          <Input
                            type="number"
                            placeholder="0"
                            {...field}
                            value={field.value ?? ""}
                          />
                        </FormControl>
                        <FormMessage />
                      </FormItem>
                    )}
                  />
                </div>
              </CardContent>
            </Card>

            <Card>
              <CardContent className="pt-6">
                <HooksEditor
                  type="server"
                  value={hooks}
                  onChange={setHooks}
                />
              </CardContent>
            </Card>
          </>
        )}

        <div className="flex items-center gap-3">
          <Button type="submit" disabled={saving}>
            {saving ? (
              <>
                <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                {isEditing ? "Saving..." : "Creating..."}
              </>
            ) : isEditing ? (
              "Save Changes"
            ) : (
              "Create Server"
            )}
          </Button>
          <Button
            type="button"
            variant="outline"
            onClick={() => navigate(-1)}
          >
            Cancel
          </Button>
        </div>
      </form>
    </Form>
  );
}

// Tag-like input for DNS entries or IPs - reusable
interface TagInputProps {
  value: string[];
  onChange: (tags: string[]) => void;
  placeholder?: string;
}

export function TagInput({ value, onChange, placeholder }: TagInputProps) {
  const [inputValue, setInputValue] = useState("");

  const addTag = () => {
    const trimmed = inputValue.trim();
    if (trimmed && !value.includes(trimmed)) {
      onChange([...value, trimmed]);
      setInputValue("");
    }
  };

  const removeTag = (index: number) => {
    const updated = [...value];
    updated.splice(index, 1);
    onChange(updated);
  };

  return (
    <div className="flex flex-col gap-2">
      <div className="flex flex-wrap gap-1.5">
        {value.map((tag, i) => (
          <Badge key={i} variant="secondary" className="gap-1 font-mono text-xs">
            {tag}
            <button
              type="button"
              onClick={() => removeTag(i)}
              className="ml-0.5 hover:text-destructive"
              aria-label={`Remove ${tag}`}
            >
              <X className="h-3 w-3" />
            </button>
          </Badge>
        ))}
      </div>
      <Input
        placeholder={placeholder}
        value={inputValue}
        onChange={(e) => setInputValue(e.target.value)}
        onKeyDown={(e) => {
          if (e.key === "Enter") {
            e.preventDefault();
            addTag();
          }
          if (e.key === "," || e.key === " ") {
            e.preventDefault();
            addTag();
          }
        }}
        onBlur={addTag}
      />
    </div>
  );
}
