import { test, expect } from "@playwright/test";

test.describe("Page loading", () => {
  test("home page loads", async ({ page }) => {
    await page.goto("/");
    await expect(page).toHaveURL("/");
    await expect(page.locator("nav")).toBeVisible();
  });

  test("about page loads", async ({ page }) => {
    await page.goto("/about");
    await expect(page).toHaveURL("/about");
    await expect(page.getByText("digital curator")).toBeVisible();
  });

  test("login page loads", async ({ page }) => {
    await page.goto("/login");
    await expect(page.locator("#username")).toBeVisible();
    await expect(page.locator("#password")).toBeVisible();
    await expect(page.locator("#login-button")).toBeVisible();
  });

  test("register page loads", async ({ page }) => {
    await page.goto("/register");
    await expect(page.locator("#username")).toBeVisible();
    await expect(page.locator("#email")).toBeVisible();
    await expect(page.locator("#password")).toBeVisible();
    await expect(page.locator("#password2")).toBeVisible();
    await expect(page.locator("#register-button")).toBeVisible();
  });
});

test.describe("Navigation", () => {
  test("nav links work", async ({ page }) => {
    await page.goto("/");

    await page.click("#nav-login");
    await expect(page).toHaveURL("/login");

    await page.click("#nav-register");
    await expect(page).toHaveURL("/register");
  });
});
