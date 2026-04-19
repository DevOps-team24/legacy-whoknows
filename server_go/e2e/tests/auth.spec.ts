import { test, expect } from "@playwright/test";

test.describe("Authentication", () => {
  test("register a new user redirects to login", async ({ page }) => {
    const uniqueUser = `testuser_${Date.now()}_${Math.floor(Math.random() * 1e6)}`;
    await page.goto("/register");

    await page.fill("#username", uniqueUser);
    await page.fill("#email", `${uniqueUser}@test.com`);
    await page.fill("#password", "testpassword123");
    await page.fill("#password2", "testpassword123");
    await page.click("#register-button");

    await expect(page).toHaveURL("/login");
    await expect(page.locator(".flashes")).toContainText("registered");
  });

  test("login redirects to home, logout redirects back to home", async ({
    page,
  }) => {
    // Register first
    const user = `logintest_${Date.now()}`;
    await page.goto("/register");
    await page.fill("#username", user);
    await page.fill("#email", `${user}@test.com`);
    await page.fill("#password", "testpassword123");
    await page.fill("#password2", "testpassword123");
    await page.click("#register-button");
    await expect(page).toHaveURL("/login");

    // Login
    await page.fill("#username", user);
    await page.fill("#password", "testpassword123");
    await page.click("#login-button");

    // Should land on the home page, logged in
    await expect(page).toHaveURL("/");
    await expect(page.locator("#nav-logout")).toBeVisible();
    await expect(page.locator("#nav-logout")).toContainText(user);

    // Logout → home, logged out
    await page.click("#nav-logout");
    await expect(page).toHaveURL("/");
    await expect(page.locator("#nav-login")).toBeVisible();
  });

  test("login with wrong password flashes an error", async ({ page }) => {
    await page.goto("/login");
    await page.fill("#username", "nonexistent");
    await page.fill("#password", "wrongpassword");
    await page.click("#login-button");

    await expect(page).toHaveURL("/login");
    await expect(page.locator(".flashes")).toContainText("Invalid");
  });
});
