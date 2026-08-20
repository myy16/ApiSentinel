"use client";

import React, { useState } from "react";
import Link from "next/link";
import { useRouter } from "next/navigation";
import { useAuth } from "../../../hooks/useAuth";
import { Shield, Lock, Mail, Building2, ArrowRight, AlertCircle, Loader2 } from "lucide-react";

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
      router.push("/dashboard/projects");
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
          <Link href="/" className="flex items-center gap-2 text-2xl font-bold tracking-tight">
            <div className="flex h-10 w-10 items-center justify-center rounded-xl bg-primary/20 text-primary">
              <Shield className="h-6 w-6" />
            </div>
            <span>ApiSentinel</span>
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
