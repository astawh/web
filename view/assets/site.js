import { initAll } from "./govuk.js";

// Keep all menus usable through ordinary links if enhancement cannot load.
window.GOVUK ??= {};
window.GOVUK.Modules ??= {};
await import("./super-navigation.js");

document.body.classList.add("js-enabled", "govuk-frontend-supported");
initAll();
const navigation = document.querySelector('[data-module="super-navigation-mega-menu"]');
if (navigation) {
  new window.GOVUK.Modules.SuperNavigationMegaMenu(navigation).init();
}
