"use client";

import React, { useState, useRef, useEffect } from "react";
import { useTheme } from "../contexts/ThemeContext";
import { Sun, Moon, Laptop, Check } from "lucide-react";

export function ThemeToggle() {
  const { theme, resolvedTheme, setTheme, toggleTheme } = useTheme();
  const [isOpen, setIsOpen] = useState(false);
  const dropdownRef = useRef<HTMLDivElement>(null);

  // Close dropdown on outside click
  useEffect(() => {
    function handleClickOutside(event: MouseEvent) {
      if (dropdownRef.current && !dropdownRef.current.contains(event.target as Node)) {
        setIsOpen(false);
      }
    }
    document.addEventListener("mousedown", handleClickOutside);
    return () => document.removeEventListener("mousedown", handleClickOutside);
  }, []);

  return (
    <div className="relative" ref={dropdownRef}>
      <button
        type="button"
        onClick={() => setIsOpen((prev) => !prev)}
        className="flex h-9 w-9 items-center justify-center rounded-xl border border-border bg-card text-foreground shadow-sm hover:border-primary/50 hover:bg-secondary/60 transition focus:outline-none focus:ring-2 focus:ring-primary"
        title={`Tema: ${theme === "system" ? "Sistem" : theme === "dark" ? "Karanlık" : "Aydınlık"} (Değiştirmek için tıkla)`}
        aria-label="Tema Seçici"
      >
        {resolvedTheme === "dark" ? (
          <Moon className="h-4 w-4 text-sky-400 transition-transform duration-300 rotate-0" />
        ) : (
          <Sun className="h-4 w-4 text-amber-500 transition-transform duration-300 rotate-0" />
        )}
      </button>

      {isOpen && (
        <div className="absolute right-0 mt-2 w-36 rounded-xl border border-border bg-card p-1.5 shadow-xl backdrop-blur-md z-50 animate-in fade-in slide-in-from-top-2 duration-150">
          <button
            type="button"
            onClick={() => {
              setTheme("light");
              setIsOpen(false);
            }}
            className={`flex w-full items-center justify-between rounded-lg px-2.5 py-1.5 text-xs font-medium transition ${
              theme === "light"
                ? "bg-primary text-primary-foreground font-bold"
                : "text-foreground hover:bg-secondary"
            }`}
          >
            <div className="flex items-center gap-2">
              <Sun className="h-3.5 w-3.5 text-amber-500" />
              <span>Aydınlık</span>
            </div>
            {theme === "light" && <Check className="h-3.5 w-3.5" />}
          </button>

          <button
            type="button"
            onClick={() => {
              setTheme("dark");
              setIsOpen(false);
            }}
            className={`flex w-full items-center justify-between rounded-lg px-2.5 py-1.5 text-xs font-medium transition ${
              theme === "dark"
                ? "bg-primary text-primary-foreground font-bold"
                : "text-foreground hover:bg-secondary"
            }`}
          >
            <div className="flex items-center gap-2">
              <Moon className="h-3.5 w-3.5 text-sky-400" />
              <span>Karanlık</span>
            </div>
            {theme === "dark" && <Check className="h-3.5 w-3.5" />}
          </button>

          <button
            type="button"
            onClick={() => {
              setTheme("system");
              setIsOpen(false);
            }}
            className={`flex w-full items-center justify-between rounded-lg px-2.5 py-1.5 text-xs font-medium transition ${
              theme === "system"
                ? "bg-primary text-primary-foreground font-bold"
                : "text-foreground hover:bg-secondary"
            }`}
          >
            <div className="flex items-center gap-2">
              <Laptop className="h-3.5 w-3.5 text-muted-foreground" />
              <span>Sistem</span>
            </div>
            {theme === "system" && <Check className="h-3.5 w-3.5" />}
          </button>
        </div>
      )}
    </div>
  );
}
