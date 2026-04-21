import { test, expect } from "@playwright/test";

test.describe("Search", () => {
  test("search page loads with input and button", async ({ page }) => {
    await page.goto("/");
    await expect(page.locator("#search-input")).toBeVisible();
  });

  test("search returns results for known query", async ({ page }) => {
    const response = await page.request.get("/api/search?q=test");
    expect(response.status()).toBe(200);
    expect(response.headers()["content-type"]).toContain("application/json");
  });

  test("search with empty query returns error", async ({ page }) => {
    const response = await page.request.get("/api/search?q=");
    expect(response.status()).not.toBe(200);
  });
});
