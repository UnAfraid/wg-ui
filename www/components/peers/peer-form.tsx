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
import { Badge } from "@/components/ui/badge";
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
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogDescription,
} from "@/components/ui/dialog";
import { HooksEditor, type PeerHookValue } from "@/components/hooks-editor";
import { CopyableText } from "@/components/copyable-text";
import {
  CREATE_PEER_MUTATION,
  UPDATE_PEER_MUTATION,
  GENERATE_WIREGUARD_KEY_MUTATION,
} from "@/lib/graphql/mutations";
import { SERVER_QUERY, PEERS_QUERY } from "@/lib/graphql/queries";
import type { Peer } from "@/lib/graphql/types";

const peerSchema = z.object({
  name: z.string().min(1, "Name is required"),
  description: z.string().optional(),
  publicKey: z.string().min(1, "Public key is required"),
  presharedKey: z.string().optional(),
  endpoint: z.string().optional(),
  allowedIPs: z.string().min(1, "At least one allowed IP is required"),
  persistentKeepalive: z.coerce
    .number()
    .int()
    .min(0)
    .max(65535)
    .optional()
    .or(z.literal("")),
});

type PeerFormValues = z.infer<typeof peerSchema>;

interface PeerFormProps {
  serverId: string;
  peer?: Peer;
}

export function PeerForm({ serverId, peer }: PeerFormProps) {
  const navigate = useNavigate();
  const isEditing = !!peer;
  const [hooks, setHooks] = useState<PeerHookValue[]>(
    peer?.hooks?.map((h) => ({
      command: h.command,
      runOnCreate: h.runOnCreate,
      runOnDelete: h.runOnDelete,
      runOnUpdate: h.runOnUpdate,
    })) ?? []
  );
  const [generatedPrivateKey, setGeneratedPrivateKey] = useState<string | null>(
    null
  );
  const [showKeyDialog, setShowKeyDialog] = useState(false);

  const [generateKey, { loading: generatingKey }] = useMutation(
    GENERATE_WIREGUARD_KEY_MUTATION
  );

  const [createPeer, { loading: creating }] = useMutation(
    CREATE_PEER_MUTATION,
    {
      refetchQueries: [
        { query: SERVER_QUERY, variables: { id: serverId } },
        { query: PEERS_QUERY },
      ],
    }
  );

  const [updatePeer, { loading: updating }] = useMutation(
    UPDATE_PEER_MUTATION,
    {
      refetchQueries: [
        { query: SERVER_QUERY, variables: { id: serverId } },
        { query: PEERS_QUERY },
      ],
    }
  );

  const form = useForm<PeerFormValues>({
    resolver: zodResolver(peerSchema),
    defaultValues: {
      name: peer?.name ?? "",
      description: peer?.description ?? "",
      publicKey: peer?.publicKey ?? "",
      presharedKey: "",
      endpoint: peer?.endpoint ?? "",
      allowedIPs: peer?.allowedIPs?.join(", ") ?? "",
      persistentKeepalive:
        peer?.persistentKeepalive ?? ("" as unknown as undefined),
    },
  });

  const handleGenerateKey = async () => {
    try {
      const { data } = await generateKey({ variables: { input: {} } });
      if (data?.generateWireguardKey) {
        form.setValue("publicKey", data.generateWireguardKey.publicKey);
        setGeneratedPrivateKey(data.generateWireguardKey.privateKey);
        setShowKeyDialog(true);
      }
    } catch (err) {
      toast.error(
        err instanceof Error ? err.message : "Failed to generate key"
      );
    }
  };

  const onSubmit = async (values: PeerFormValues) => {
    try {
      const allowedIPs = values.allowedIPs
        .split(",")
        .map((ip) => ip.trim())
        .filter(Boolean);

      if (isEditing) {
        await updatePeer({
          variables: {
            input: {
              id: peer.id,
              name: values.name,
              description: values.description || "",
              publicKey: values.publicKey,
              presharedKey: values.presharedKey || undefined,
              endpoint: values.endpoint || "",
              allowedIPs,
              persistentKeepalive: values.persistentKeepalive || undefined,
              hooks: hooks.length > 0 ? hooks : undefined,
            },
          },
        });
        toast.success("Peer updated");
      } else {
        await createPeer({
          variables: {
            input: {
              serverId,
              name: values.name,
              description: values.description || "",
              publicKey: values.publicKey,
              presharedKey: values.presharedKey || undefined,
              endpoint: values.endpoint || "",
              allowedIPs,
              persistentKeepalive: values.persistentKeepalive || undefined,
              hooks: hooks.length > 0 ? hooks : undefined,
            },
          },
        });
        toast.success("Peer created");
      }
      navigate(`/servers/${serverId}`);
    } catch (err) {
      toast.error(
        err instanceof Error ? err.message : "Failed to save peer"
      );
    }
  };

  const saving = creating || updating;

  return (
    <>
      <Form {...form}>
        <form
          onSubmit={form.handleSubmit(onSubmit)}
          className="flex flex-col gap-6"
        >
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
                      <Input placeholder="Peer name" {...field} />
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
            </CardContent>
          </Card>

          <Card>
            <CardHeader>
              <div className="flex items-center justify-between">
                <CardTitle className="text-base">Key Configuration</CardTitle>
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
                  Generate Key Pair
                </Button>
              </div>
            </CardHeader>
            <CardContent className="flex flex-col gap-4">
              <FormField
                control={form.control}
                name="publicKey"
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>Public Key</FormLabel>
                    <FormControl>
                      <Input
                        placeholder="Peer's public key"
                        className="font-mono text-sm"
                        {...field}
                      />
                    </FormControl>
                    <FormDescription>
                      {isEditing
                        ? "The peer's WireGuard public key"
                        : "Click Generate to create a new key pair, or paste an existing public key."}
                    </FormDescription>
                    <FormMessage />
                  </FormItem>
                )}
              />
              <FormField
                control={form.control}
                name="presharedKey"
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>Preshared Key (Optional)</FormLabel>
                    <FormControl>
                      <Input
                        placeholder={
                          isEditing
                            ? "Leave empty to keep current"
                            : "Optional preshared key"
                        }
                        className="font-mono text-sm"
                        {...field}
                      />
                    </FormControl>
                    <FormDescription>
                      An additional layer of symmetric-key cryptography
                    </FormDescription>
                    <FormMessage />
                  </FormItem>
                )}
              />
            </CardContent>
          </Card>

          <Card>
            <CardHeader>
              <CardTitle className="text-base">Network</CardTitle>
            </CardHeader>
            <CardContent className="flex flex-col gap-4">
              <FormField
                control={form.control}
                name="endpoint"
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>Endpoint</FormLabel>
                    <FormControl>
                      <Input
                        placeholder="192.168.1.100:51820"
                        {...field}
                      />
                    </FormControl>
                    <FormDescription>
                      {"The peer's publicly accessible address:port (optional for server-side peers)"}
                    </FormDescription>
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
                      <Input
                        placeholder="10.0.0.2/32"
                        {...field}
                      />
                    </FormControl>
                    <FormDescription>
                      Comma-separated IP ranges this peer is allowed to use
                    </FormDescription>
                    <FormMessage />
                  </FormItem>
                )}
              />
              <FormField
                control={form.control}
                name="persistentKeepalive"
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>Persistent Keepalive (seconds)</FormLabel>
                    <FormControl>
                      <Input
                        type="number"
                        placeholder="25"
                        {...field}
                        value={field.value ?? ""}
                      />
                    </FormControl>
                    <FormDescription>
                      Send keepalive every N seconds (0 to disable)
                    </FormDescription>
                    <FormMessage />
                  </FormItem>
                )}
              />
            </CardContent>
          </Card>

          <Card>
            <CardContent className="pt-6">
              <HooksEditor type="peer" value={hooks} onChange={setHooks} />
            </CardContent>
          </Card>

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
                "Create Peer"
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

      {/* Private Key Reveal Dialog */}
      <Dialog open={showKeyDialog} onOpenChange={setShowKeyDialog}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Private Key Generated</DialogTitle>
            <DialogDescription>
              Save this private key now. It will only be shown once and is needed
              for the peer to connect. The public key has been set in the form.
            </DialogDescription>
          </DialogHeader>
          <div className="flex flex-col gap-3">
            <div>
              <p className="mb-1 text-xs font-medium text-muted-foreground">
                Private Key (save this!)
              </p>
              <CopyableText
                text={generatedPrivateKey ?? ""}
                truncate={false}
                className="rounded-md border border-border bg-muted p-2"
              />
            </div>
            <div>
              <p className="mb-1 text-xs font-medium text-muted-foreground">
                Public Key (set in form)
              </p>
              <CopyableText
                text={form.getValues("publicKey")}
                truncate={false}
                className="rounded-md border border-border bg-muted p-2"
              />
            </div>
          </div>
          <Button onClick={() => setShowKeyDialog(false)} className="mt-2">
            I have saved the private key
          </Button>
        </DialogContent>
      </Dialog>
    </>
  );
}
