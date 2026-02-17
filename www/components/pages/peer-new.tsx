"use client";

import { Link } from "react-router-dom";
import { ArrowLeft } from "lucide-react";

import { Button } from "@/components/ui/button";
import { PeerForm } from "@/components/peers/peer-form";

export default function NewPeerPage({ id }: { id: string }) {

  return (
    <div className="flex flex-col gap-6">
      <div className="flex items-center gap-3">
        <Button variant="ghost" size="icon" className="h-8 w-8" asChild>
          <Link to={`/servers/${id}`}>
            <ArrowLeft className="h-4 w-4" />
          </Link>
        </Button>
        <div>
          <h1 className="text-2xl font-semibold tracking-tight text-foreground">
            New Peer
          </h1>
          <p className="mt-0.5 text-sm text-muted-foreground">
            Add a peer to this server
          </p>
        </div>
      </div>

      <PeerForm serverId={id} />
    </div>
  );
}
