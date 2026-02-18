"use client";

import { useMemo, useState } from "react";
import { Copy, Download, QrCode } from "lucide-react";
import { toast } from "sonner";
import QRCode from "react-qr-code";

import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Textarea } from "@/components/ui/textarea";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from "@/components/ui/dialog";
import type { Peer, Server } from "@/lib/graphql/types";

const DEFAULT_ALLOWED_IPS = "0.0.0.0/0, ::/0";

function normalizeCSV(value: string): string {
  return value
    .split(",")
    .map((entry) => entry.trim())
    .filter(Boolean)
    .join(", ");
}

function defaultEndpoint(listenPort?: number | null): string {
  if (typeof window === "undefined") {
    return "";
  }

  const host = window.location.hostname;
  if (!host) {
    return "";
  }
  if (!listenPort || listenPort <= 0) {
    return host;
  }
  return `${host}:${listenPort}`;
}

function buildClientConfig(params: {
  peer: Peer;
  server: Server;
  privateKey: string;
  addresses: string;
  dns: string;
  endpoint: string;
  allowedIPs: string;
  persistentKeepalive: string;
  mtu: string;
}): string {
  const {
    peer,
    server,
    privateKey,
    addresses,
    dns,
    endpoint,
    allowedIPs,
    persistentKeepalive,
    mtu,
  } = params;

  const normalizedAddresses = normalizeCSV(addresses);
  const normalizedDNS = normalizeCSV(dns);
  const normalizedAllowedIPs = normalizeCSV(allowedIPs);
  const normalizedEndpoint = endpoint.trim();
  const normalizedPrivateKey = privateKey.trim();
  const normalizedPresharedKey = (peer.presharedKey || "").trim();

  const keepaliveValue = Number.parseInt(persistentKeepalive.trim(), 10);
  const mtuValue = Number.parseInt(mtu.trim(), 10);

  const lines: string[] = [
    `[Interface]`,
    `PrivateKey = ${normalizedPrivateKey || "<paste-client-private-key>"}`,
  ];

  if (normalizedAddresses) {
    lines.push(`Address = ${normalizedAddresses}`);
  }
  if (normalizedDNS) {
    lines.push(`DNS = ${normalizedDNS}`);
  }
  if (!Number.isNaN(mtuValue) && mtuValue > 0) {
    lines.push(`MTU = ${mtuValue}`);
  }

  lines.push("", "[Peer]", `PublicKey = ${server.publicKey}`);

  if (normalizedPresharedKey) {
    lines.push(`PresharedKey = ${normalizedPresharedKey}`);
  }
  if (normalizedEndpoint) {
    lines.push(`Endpoint = ${normalizedEndpoint}`);
  }
  if (normalizedAllowedIPs) {
    lines.push(`AllowedIPs = ${normalizedAllowedIPs}`);
  }
  if (!Number.isNaN(keepaliveValue) && keepaliveValue > 0) {
    lines.push(`PersistentKeepalive = ${keepaliveValue}`);
  }

  return `${lines.join("\n")}\n`;
}

interface PeerClientConfigDialogProps {
  server: Server;
  peer: Peer;
  trigger: React.ReactNode;
}

export function PeerClientConfigDialog({
  server,
  peer,
  trigger,
}: PeerClientConfigDialogProps) {
  const [open, setOpen] = useState(false);
  const [privateKey, setPrivateKey] = useState("");
  const [addresses, setAddresses] = useState((peer.allowedIPs ?? []).join(", "));
  const [dns, setDns] = useState((server.dns ?? []).join(", "));
  const [endpoint, setEndpoint] = useState("");
  const [allowedIPs, setAllowedIPs] = useState(DEFAULT_ALLOWED_IPS);
  const [persistentKeepalive, setPersistentKeepalive] = useState("25");
  const [mtu, setMtu] = useState(server.mtu > 0 ? String(server.mtu) : "");
  const [showQr, setShowQr] = useState(false);

  const config = useMemo(
    () =>
      buildClientConfig({
        peer,
        server,
        privateKey,
        addresses,
        dns,
        endpoint,
        allowedIPs,
        persistentKeepalive,
        mtu,
      }),
    [
      peer,
      server,
      privateKey,
      addresses,
      dns,
      endpoint,
      allowedIPs,
      persistentKeepalive,
      mtu,
    ]
  );

  const resetForm = () => {
    setPrivateKey("");
    setAddresses((peer.allowedIPs ?? []).join(", "));
    setDns((server.dns ?? []).join(", "));
    setEndpoint(defaultEndpoint(server.listenPort));
    setAllowedIPs(DEFAULT_ALLOWED_IPS);
    setPersistentKeepalive("25");
    setMtu(server.mtu > 0 ? String(server.mtu) : "");
    setShowQr(false);
  };

  const handleOpenChange = (next: boolean) => {
    setOpen(next);
    if (next) {
      resetForm();
    }
  };

  const handleCopy = async () => {
    try {
      await navigator.clipboard.writeText(config);
      toast.success("Config copied to clipboard");
    } catch (err) {
      toast.error(err instanceof Error ? err.message : "Failed to copy config");
    }
  };

  const handleDownload = () => {
    const blob = new Blob([config], { type: "text/plain;charset=utf-8" });
    const url = URL.createObjectURL(blob);
    const link = document.createElement("a");
    link.href = url;
    link.download = `${peer.name || "peer"}.conf`;
    link.click();
    URL.revokeObjectURL(url);
  };

  const handleGenerateQr = () => {
    if (!privateKey.trim()) {
      toast.error("Client private key is required before generating QR");
      return;
    }
    setShowQr(true);
  };

  return (
    <Dialog open={open} onOpenChange={handleOpenChange}>
      <DialogTrigger asChild>{trigger}</DialogTrigger>
      <DialogContent className="max-h-[90vh] overflow-y-auto sm:max-w-3xl">
        <DialogHeader>
          <DialogTitle>Client Config: {peer.name}</DialogTitle>
          <DialogDescription>
            Generate a WireGuard client config file and optional QR code.
          </DialogDescription>
        </DialogHeader>

        <div className="grid grid-cols-1 gap-4 md:grid-cols-2">
          <div className="flex flex-col gap-1.5">
            <Label>Client Private Key</Label>
            <Input
              value={privateKey}
              onChange={(e) => setPrivateKey(e.target.value)}
              placeholder="Paste client private key"
              className="font-mono text-xs"
            />
          </div>
          <div className="flex flex-col gap-1.5">
            <Label>Server Endpoint</Label>
            <Input
              value={endpoint}
              onChange={(e) => setEndpoint(e.target.value)}
              placeholder="vpn.example.com:51820"
              className="font-mono text-xs"
            />
          </div>
          <div className="flex flex-col gap-1.5">
            <Label>Client Address(es)</Label>
            <Input
              value={addresses}
              onChange={(e) => setAddresses(e.target.value)}
              placeholder="10.0.0.2/32"
              className="font-mono text-xs"
            />
          </div>
          <div className="flex flex-col gap-1.5">
            <Label>Peer Allowed IPs</Label>
            <Input
              value={allowedIPs}
              onChange={(e) => setAllowedIPs(e.target.value)}
              placeholder="0.0.0.0/0, ::/0"
              className="font-mono text-xs"
            />
          </div>
          <div className="flex flex-col gap-1.5">
            <Label>DNS</Label>
            <Input
              value={dns}
              onChange={(e) => setDns(e.target.value)}
              placeholder="1.1.1.1, 8.8.8.8"
              className="font-mono text-xs"
            />
          </div>
          <div className="grid grid-cols-2 gap-3">
            <div className="flex flex-col gap-1.5">
              <Label>MTU</Label>
              <Input
                value={mtu}
                onChange={(e) => setMtu(e.target.value)}
                placeholder="1420"
                className="font-mono text-xs"
              />
            </div>
            <div className="flex flex-col gap-1.5">
              <Label>Keepalive</Label>
              <Input
                value={persistentKeepalive}
                onChange={(e) => setPersistentKeepalive(e.target.value)}
                placeholder="25"
                className="font-mono text-xs"
              />
            </div>
          </div>
        </div>

        <div className="flex flex-wrap items-center gap-2">
          <Button type="button" variant="outline" size="sm" onClick={handleCopy}>
            <Copy className="mr-1.5 h-3.5 w-3.5" />
            Copy
          </Button>
          <Button
            type="button"
            variant="outline"
            size="sm"
            onClick={handleDownload}
          >
            <Download className="mr-1.5 h-3.5 w-3.5" />
            Download
          </Button>
          <Button
            type="button"
            variant="outline"
            size="sm"
            onClick={handleGenerateQr}
          >
            <QrCode className="mr-1.5 h-3.5 w-3.5" />
            Generate QR
          </Button>
        </div>

        <div className="flex flex-col gap-1.5">
          <Label>Generated Config</Label>
          <Textarea
            readOnly
            value={config}
            rows={12}
            className="font-mono text-xs"
          />
        </div>

        {showQr && (
          <div className="flex flex-col gap-1.5">
            <Label>QR Code</Label>
            <div className="rounded-md border bg-white p-4">
              <div className="mx-auto w-fit">
                <QRCode value={config} size={256} />
              </div>
            </div>
          </div>
        )}
      </DialogContent>
    </Dialog>
  );
}
