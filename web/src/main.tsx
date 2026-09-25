import { bootstrapAppearance, restoreCachedAppearance } from "@/services/appearance-bootstrap";
import { isAuthPath, isRootPath } from "@/lib/public-shell";

restoreCachedAppearance();
void bootstrapAppearance();

const path = window.location.pathname;

if (/^\/welcome\/?$/.test(path)) {
    void import("./welcome-application");
} else if (isRootPath(path) || isAuthPath(path)) {
    void import("./public-application");
} else {
    void import("./application");
}
