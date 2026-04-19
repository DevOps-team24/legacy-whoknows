import { test, expect } from "@playwright/test";

const uniqueUser = `testuser_${Date.now()}`;

test.describe("Authentication", () => {
  test("register a new user", async ({ page }) => {
    await page.goto("/register");

    await page.fill("#username", uniqueUser);
    await page.fill("#email", `${uniqueUser}@test.com`);
    await page.fill("#password", "testpassword123");
    await page.fill("#password2", "testpassword123");
    await page.click("#register-button");

    // Server returns JSON with success message
    await page.waitForURL("**/api/register");
    await expect(page.locator("body")).toContainText("registered");
  });

  test("login and logout", async ({ page }) => {
    // Register first via the form
    const user = `logintest_${Date.now()}`;
    await page.goto("/register");
    await page.fill("#username", user);
    await page.fill("#email", `${user}@test.com`);
    await page.fill("#password", "testpassword123");
    await page.fill("#password2", "testpassword123");
    await page.click("#register-button");
    await page.waitForURL("**/api/register");

    // Login via the form
    await page.goto("/login");
    await page.fill("#username", user);
    await page.fill("#password", "testpassword123");
    await page.click("#login-button");

    // Server returns JSON success
    await page.waitForURL("**/api/login");
    await expect(page.locator("body")).toContainText("logged in");

    // Navigate to home — should see logout link
    await page.goto("/");
    await expect(page.locator("#nav-logout")).toBeVisible();
    await expect(page.locator("#nav-logout")).toContainText(user);

    // Logout
    await page.click("#nav-logout");
    await expect(page.locator("body")).toContainText("logged out");

    // Navigate home — should see login link again
    await page.goto("/");
    await expect(page.locator("#nav-login")).toBeVisible();
  });

  test("login with wrong password shows error", async ({ page }) => {
    await page.goto("/login");
    await page.fill("#username", "nonexistent");
    await page.fill("#password", "wrongpassword");
    await page.click("#login-button");

    await page.waitForURL("**/api/login");
    await expect(page.locator("body")).toContainText("Invalid");
  });
});
