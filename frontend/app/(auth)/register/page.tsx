"use client";

import React, { useState } from "react";
import Link from "next/link";
import { useRouter } from "next/navigation";
import { useAuth } from "../../../hooks/useAuth";
import Image from "next/image";
import { Lock, Mail, Building2, ArrowRight, AlertCircle, Loader2 } from "lucide-react";

export default function RegisterPage() {
  const router = useRouter();
  const { register } = useAuth();

  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [orgName, setOrgName] = useState("");
  const [error, setError] = useState<string | null>(null);
  const [isSubmitting, setIsSubmitting] = useState(false);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setError(null);

    if (password.length < 8) {
      setError("Şifre en az 8 karakter olmalıdır.");
      return;
    }

    setIsSubmitting(true);

    try {
      await register(email, password, orgName || undefined);
      router.push("/projects");
    } catch (err: any) {
      setError(err.message || "Kayıt başarısız. Lütfen bilgilerinizi kontrol ediniz.");
    } finally {
      setIsSubmitting(false);
    }
  };

  return (
    <div className="flex min-h-screen flex-col items-center justify-center bg-background px-6 py-12">
      <div className="w-full max-w-md space-y-8">
        {/* Header */}
        <div className="flex flex-col items-center text-center">
          <Link href="/" className="group flex flex-col items-center gap-3">
            <div className="relative flex h-16 w-16 items-center justify-center rounded-2xl bg-secondary/60 border border-border/80 p-2.5 shadow-md shadow-primary/5 ring-1 ring-border/50 group-hover:border-primary/50 group-hover:bg-secondary/90 transition-all duration-300">
              <Image
                src="/logo-emblem.png"
                alt="ApiSentinel Emblem"
                width={56}
                height={56}
                className="h-full w-full object-contain drop-shadow-[0_2px_8px_rgba(37,99,235,0.4)]"
                priority
              />
            </div>
            <div className="flex flex-col items-center">
              <span className="text-2xl font-black tracking-tight text-foreground">ApiSentinel</span>
              <span className="text-[10px] font-mono uppercase tracking-widest text-muted-foreground">
                Security Gateway
              </span>
            </div>
          </Link>
          <h2 className="mt-6 text-2xl font-bold tracking-tight">Yeni Hesap Oluşturun</h2>
          <p className="mt-2 text-sm text-muted-foreground">
            Geliştirici güvenliği platformuna hemen katılın
          </p>
        </div>

        {/* Error Alert */}
        {error && (
          <div className="flex items-center gap-2 rounded-lg border border-destructive/50 bg-destructive/10 p-3 text-sm text-destructive">
            <AlertCircle className="h-4 w-4 shrink-0" />
            <span>{error}</span>
          </div>
        )}

        {/* Form */}
        <form onSubmit={handleSubmit} className="mt-8 space-y-4 rounded-xl border border-border bg-card p-6 shadow-sm">
          <div>
            <label className="block text-xs font-semibold uppercase tracking-wider text-muted-foreground mb-1.5">
              E-posta Adresi
            </label>
            <div className="relative flex items-center">
              <Mail className="absolute left-3 h-4 w-4 text-muted-foreground" />
              <input
                type="email"
                required
                value={email}
                onChange={(e) => setEmail(e.target.value)}
                placeholder="developer@example.com"
                className="w-full rounded-lg border border-input bg-background/50 py-2 pl-9 pr-3 text-sm placeholder:text-muted-foreground/60 focus:border-primary focus:outline-none focus:ring-1 focus:ring-primary"
              />
            </div>
          </div>

          <div>
            <label className="block text-xs font-semibold uppercase tracking-wider text-muted-foreground mb-1.5">
              Organizasyon Adı (Opsiyonel)
            </label>
            <div className="relative flex items-center">
              <Building2 className="absolute left-3 h-4 w-4 text-muted-foreground" />
              <input
                type="text"
                value={orgName}
                onChange={(e) => setOrgName(e.target.value)}
                placeholder="Örn: Acme Tech"
                className="w-full rounded-lg border border-input bg-background/50 py-2 pl-9 pr-3 text-sm placeholder:text-muted-foreground/60 focus:border-primary focus:outline-none focus:ring-1 focus:ring-primary"
              />
            </div>
          </div>

          <div>
            <label className="block text-xs font-semibold uppercase tracking-wider text-muted-foreground mb-1.5">
              Şifre (Min. 8 Karakter)
            </label>
            <div className="relative flex items-center">
              <Lock className="absolute left-3 h-4 w-4 text-muted-foreground" />
              <input
                type="password"
                required
                value={password}
                onChange={(e) => setPassword(e.target.value)}
                placeholder="••••••••"
                className="w-full rounded-lg border border-input bg-background/50 py-2 pl-9 pr-3 text-sm placeholder:text-muted-foreground/60 focus:border-primary focus:outline-none focus:ring-1 focus:ring-primary"
              />
            </div>
          </div>

          <button
            type="submit"
            disabled={isSubmitting}
            className="flex w-full items-center justify-center gap-2 rounded-lg bg-primary py-2.5 text-sm font-semibold text-primary-foreground shadow-sm transition hover:bg-primary/90 disabled:opacity-50"
          >
            {isSubmitting ? (
              <>
                <Loader2 className="h-4 w-4 animate-spin" />
                <span>Hesap Oluşturuluyor...</span>
              </>
            ) : (
              <>
                <span>Hesap Oluştur</span>
                <ArrowRight className="h-4 w-4" />
              </>
            )}
          </button>
        </form>

        <p className="text-center text-sm text-muted-foreground">
          Zaten bir hesabınız var mı?{" "}
          <Link href="/login" className="font-semibold text-primary hover:underline">
            Giriş Yapın
          </Link>
        </p>
      </div>
    </div>
  );
}
