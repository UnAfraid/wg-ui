"use client";

import { useQuery } from "@apollo/client";
import { Link } from "react-router-dom";
import { ArrowLeft } from "lucide-react";

import { Button } from "@/components/ui/button";
import { Skeleton } from "@/components/ui/skeleton";
import { PeerForm } from "@/components/peers/peer-form";
import { PEER_QUERY } from "@/lib/graphql/queries";
import type { Peer } from "@/lib/graphql/types";

export default function EditPeerPage({
  id,
  peerId,
}: {
  id: string;
  peerId: string;
}) {
  const { data, loading, error } = useQuery(PEER_QUERY, {
    variables: { id: peerId },
  });

  const peer: Peer | null = data?.node ?? null;

  if (loading) {
    return (
      <div className="flex flex-col gap-6">
        <Skeleton className="h-8 w-64" />
        <Skeleton className="h-96 w-full" />
      </div>
    );
  }

  if (error || !peer) {
    return (
      <div className="flex flex-col items-center justify-center py-16">
        <p className="text-sm text-muted-foreground">Peer not found.</p>
        <Button variant="outline" className="mt-4" asChild>
          <Link to={`/servers/${id}`}>Back to Server</Link>
        </Button>
      </div>
    );
  }

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
            Edit {peer.name}
          </h1>
          <p className="mt-0.5 text-sm text-muted-foreground">
            Update peer configuration
          </p>
        </div>
      </div>

      <PeerForm serverId={id} peer={peer} />
    </div>
  );
}
