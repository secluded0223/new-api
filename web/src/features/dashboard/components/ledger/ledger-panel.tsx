/*
Copyright (C) 2023-2026 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
GNU Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License
along with this program. If not, see <https://www.gnu.org/licenses/>.

For commercial licensing, please contact support@quantumnous.com
*/
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { Plus } from "lucide-react";
import { useMemo, useState } from "react";
import { useTranslation } from "react-i18next";

import {
  Drawer,
  DrawerContent,
  DrawerFooter,
  DrawerHeader,
  DrawerTitle,
} from "@/components/ui/drawer";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { api } from "@/lib/api";

import { ledgerListGridColumns } from "./layout";

type Expense = {
  id: number;
  occurred_at: number;
  category: string;
  amount: number;
  currency: string;
  note: string;
};
type LedgerSummary = {
  revenue_cents: number;
  expenses: number;
  profit: number;
  currency: string;
  expenses_list: Expense[];
  trend: {
    date: string;
    revenue_cents: number;
    expense_cents: number;
    profit_cents: number;
  }[];
};

function formatMoney(cents: number, _currency?: string) {
  return new Intl.NumberFormat(undefined, {
    minimumFractionDigits: 2,
    maximumFractionDigits: 2,
  }).format(cents / 100);
}
function formatDate(date: Date) {
  return `${date.getFullYear()}-${String(date.getMonth() + 1).padStart(2, "0")}-${String(date.getDate()).padStart(2, "0")}`;
}
function toTimestamp(date: string) {
  return Math.floor(new Date(`${date}T00:00:00`).getTime() / 1000);
}

export function LedgerPanel() {
  const { t } = useTranslation();
  const queryClient = useQueryClient();
  const today = formatDate(new Date());
  const [startDate, setStartDate] = useState(today);
  const [endDate, setEndDate] = useState(today);
  const [period, setPeriod] = useState("1");
  const [drawerOpen, setDrawerOpen] = useState(false);
  const [ledgerView, setLedgerView] = useState("trend");
  const category = "upstream_recharge";
  const [amount, setAmount] = useState("");
  const [occurredAt, setOccurredAt] = useState(today);
  const [note, setNote] = useState("");
  const range = useMemo(
    () => ({
      start_timestamp: period === "all" ? -1 : toTimestamp(startDate),
      end_timestamp:
        period === "all"
          ? Math.floor(Date.now() / 1000)
          : toTimestamp(endDate) + 86399,
    }),
    [startDate, endDate, period],
  );
  const ledgerQuery = useQuery({
    queryKey: ["ledger", range],
    queryFn: async () =>
      (await api.get<{ data: LedgerSummary }>("/api/ledger", { params: range }))
        .data.data,
  });
  const addExpense = useMutation({
    mutationFn: async () => {
      const parsedAmount = Number(amount);
      if (
        !category.trim() ||
        !Number.isFinite(parsedAmount) ||
        parsedAmount <= 0
      ) {
        throw new Error(t("Enter a category and a valid amount"));
      }
      return api.post("/api/ledger/expenses", {
        occurred_at: toTimestamp(occurredAt),
        category: category.trim(),
        amount: Math.round(parsedAmount * 100),
        currency: "USD",
        note: note.trim(),
      });
    },
    onSuccess: async () => {
      setAmount("");
      setNote("");
      setDrawerOpen(false);
      await queryClient.invalidateQueries({ queryKey: ["ledger"] });
    },
  });
  const applyPeriod = (value: string) => {
    const now = new Date();
    const end = formatDate(now);
    if (value === "all") {
      setPeriod(value);
      setStartDate("");
      setEndDate(end);
      return;
    }
    const days = Number(value);
    const start = formatDate(
      new Date(now.getFullYear(), now.getMonth(), now.getDate() - days + 1),
    );
    setPeriod(value);
    setStartDate(start);
    setEndDate(end);
  };
  const summary = ledgerQuery.data;
  const trend = summary?.trend ?? [];
  return (
    <div className="space-y-4">
      <div className="flex flex-wrap items-center gap-3 rounded-lg border p-3">
        <div className="order-last ml-auto flex shrink-0 items-center gap-1">
          {[
            ["1", t("1 Day")],
            ["7", t("7 Days")],
            ["15", t("15 Days")],
            ["30", t("30 Days")],
            ["all", t("Total")],
          ].map(([value, label]) => (
            <Button
              key={value}
              size="sm"
              variant={period === value ? "default" : "outline"}
              aria-pressed={period === value}
              onClick={() => applyPeriod(value)}
            >
              {label}
            </Button>
          ))}
        </div>
        <div className="order-first flex shrink-0 items-center gap-2">
          <label className="shrink-0">
            <Input
              aria-label={t("From")}
              className="h-8 w-36 text-xs sm:w-40"
              type="date"
              value={startDate}
              disabled={period === "all"}
              onChange={(event) => {
                setStartDate(event.target.value);
                setPeriod("");
              }}
            />
          </label>
          <span className="text-muted-foreground">-</span>
          <label className="shrink-0">
            <Input
              aria-label={t("To")}
              className="h-8 w-36 text-xs sm:w-40"
              type="date"
              value={endDate}
              disabled={period === "all"}
              onChange={(event) => {
                setEndDate(event.target.value);
                setPeriod("");
              }}
            />
          </label>
        </div>
        <div className="order-last flex shrink-0 gap-1">
          <Drawer
            open={drawerOpen}
            direction="right"
            onOpenChange={setDrawerOpen}
          >
            <Button className="ml-auto" onClick={() => setDrawerOpen(true)}>
              <Plus className="size-4" aria-hidden="true" />
              {t("Add expense")}
            </Button>
            <DrawerContent className="w-full sm:max-w-md">
              <DrawerHeader>
                <DrawerTitle>{t("Add expense")}</DrawerTitle>
              </DrawerHeader>
              <div className="grid gap-3 px-4">
                <label className="grid gap-1 text-sm">
                  {t("Date")}
                  <Input
                    type="date"
                    value={occurredAt}
                    onChange={(event) => setOccurredAt(event.target.value)}
                  />
                </label>
                <div className="grid gap-1 text-sm">
                  <span>{t("Category")}</span>
                  <div className="border-input bg-muted/30 text-muted-foreground flex h-8 items-center rounded-lg border px-2.5">
                    {t("Upstream recharge")}
                  </div>
                </div>
                <label className="grid gap-1 text-sm">
                  {t("Amount (USD)")}
                  <Input
                    type="number"
                    min="0.01"
                    step="0.01"
                    value={amount}
                    onChange={(event) => setAmount(event.target.value)}
                  />
                </label>
                <label className="grid gap-1 text-sm">
                  {t("Note")}
                  <Input
                    value={note}
                    onChange={(event) => setNote(event.target.value)}
                  />
                </label>
                {addExpense.error && (
                  <p className="text-destructive text-sm">
                    {addExpense.error.message}
                  </p>
                )}
              </div>
              <DrawerFooter>
                <Button
                  onClick={() => addExpense.mutate()}
                  disabled={addExpense.isPending}
                >
                  <Plus className="size-4" />
                  {t("Add expense")}
                </Button>
                <Button variant="outline" onClick={() => setDrawerOpen(false)}>
                  {t("Cancel")}
                </Button>
              </DrawerFooter>
            </DrawerContent>
          </Drawer>
        </div>
      </div>
      <div className="grid gap-3 sm:grid-cols-3">
        <div className="rounded-lg border p-4">
          <p className="text-muted-foreground text-sm">
            {t("Consumption revenue")}
          </p>
          <p className="mt-1 text-2xl font-semibold">
            {formatMoney(
              summary?.revenue_cents ?? 0,
              summary?.currency ?? "USD",
            )}
          </p>
        </div>
        <div className="rounded-lg border p-4">
          <p className="text-muted-foreground text-sm">
            {t("Recorded expenses")}
          </p>
          <p className="mt-1 text-2xl font-semibold">
            {formatMoney(summary?.expenses ?? 0, "USD")}
          </p>
        </div>
        <div className="rounded-lg border bg-emerald-500/5 p-4">
          <p className="text-muted-foreground text-sm">{t("Net profit")}</p>
          <p className="mt-1 text-2xl font-semibold">
            {formatMoney(summary?.profit ?? 0, summary?.currency ?? "USD")}
          </p>
        </div>
      </div>
      <div className="rounded-lg border">
        <div className="flex items-center gap-2 border-b px-4 py-3 text-sm">
          <button
            type="button"
            aria-current={ledgerView === "trend" ? "page" : undefined}
            className={
              ledgerView === "trend"
                ? "text-foreground font-semibold"
                : "text-muted-foreground hover:text-foreground"
            }
            onClick={() => setLedgerView("trend")}
          >
            {t("Business trend")}
          </button>
          <span className="text-muted-foreground/60" aria-hidden="true">
            /
          </span>
          <button
            type="button"
            aria-current={ledgerView === "expenses" ? "page" : undefined}
            className={
              ledgerView === "expenses"
                ? "text-foreground font-semibold"
                : "text-muted-foreground hover:text-foreground"
            }
            onClick={() => setLedgerView("expenses")}
          >
            {t("Expense records")}
          </button>
        </div>
        {ledgerView === "expenses" ? (
          <div className="space-y-3 px-4 py-4">
            <div
              className={`text-muted-foreground grid gap-3 text-xs ${ledgerListGridColumns}`}
            >
              <span>{t("Date")}</span>
              <span>{t("Category")}</span>
              <span>{t("Note")}</span>
              <span className="text-right">{t("Amount")}</span>
            </div>
            {(summary?.expenses_list ?? []).length === 0 ? (
              <p className="text-muted-foreground py-8 text-center">
                {t("No expenses recorded")}
              </p>
            ) : (
              summary?.expenses_list.map((expense) => (
                <div
                  key={expense.id}
                  className={`grid items-center gap-3 text-sm ${ledgerListGridColumns}`}
                >
                  <span className="text-muted-foreground whitespace-nowrap tabular-nums">
                    {formatDate(new Date(expense.occurred_at * 1000)).slice(5)}
                  </span>
                  <span className="truncate">
                    {expense.category === "upstream_recharge"
                      ? t("Upstream recharge")
                      : expense.category}
                  </span>
                  <span className="text-muted-foreground min-w-0 truncate">
                    {expense.note || "-"}
                  </span>
                  <span className="text-right tabular-nums">
                    {formatMoney(expense.amount, expense.currency)}
                  </span>
                </div>
              ))
            )}
          </div>
        ) : (
          <div className="space-y-3 px-4 py-4">
            <div
              className={`text-muted-foreground grid gap-3 text-xs ${ledgerListGridColumns}`}
            >
              <span>{t("Date")}</span>
              <span>{t("Consumption revenue")}</span>
              <span>{t("Recorded expenses")}</span>
              <span>{t("Net profit")}</span>
            </div>
            {trend.length === 0 ? (
              <p className="text-muted-foreground py-8 text-center">
                {t("No data")}
              </p>
            ) : (
              trend.map((point) => (
                <div
                  key={point.date}
                  className={`grid items-center gap-3 text-sm ${ledgerListGridColumns}`}
                >
                  <span className="text-muted-foreground">
                    {point.date.slice(5)}
                  </span>
                  <div className="min-w-0">
                    <span className="tabular-nums">
                      {formatMoney(point.revenue_cents)}
                    </span>
                  </div>
                  <div className="min-w-0">
                    <span className="tabular-nums">
                      {formatMoney(point.expense_cents)}
                    </span>
                  </div>
                  <div className="min-w-0">
                    <span className="tabular-nums">
                      {formatMoney(point.profit_cents)}
                    </span>
                  </div>
                </div>
              ))
            )}
          </div>
        )}
      </div>
    </div>
  );
}
