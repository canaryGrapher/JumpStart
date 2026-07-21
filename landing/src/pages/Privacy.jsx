import LegalLayout from "./LegalLayout";

export default function Privacy() {
  return (
    <LegalLayout title="Privacy Policy" updated="July 21, 2026">
      <p>
        This Privacy Policy explains how the JumpStart project ("JumpStart", "we", "us"), a
        project of workvar.com, handles information across two things: the JumpStart desktop
        application (the "App") and the JumpStart website at jumpstart.workvar.com (the "Site").
        We built JumpStart to be local-first, and this policy reflects that. In short: your
        projects and code stay on your machine, and any data we do collect is never sold, only
        studied to improve the product.
      </p>

      <h2>The short version</h2>
      <ul>
        <li>The App stores your data locally on your computer. It is not uploaded to us.</li>
        <li>The App only reaches the network for actions you initiate (git operations, publishing releases, update checks) and, if configured, anonymous usage analytics.</li>
        <li>The Site uses privacy-conscious analytics to understand how visitors use it.</li>
        <li>We never sell your data, and we never share it with advertisers. We study it to improve JumpStart.</li>
      </ul>

      <h2>The desktop application</h2>
      <p>
        JumpStart is a native desktop app built with Wails. It is designed so your work never
        leaves your machine unless you explicitly ask it to.
      </p>
      <h3>What is stored, and where</h3>
      <p>
        Your project configuration, processes, environment variables, and Kanban board data are
        stored locally in <code>~/.jumpstart/config.json</code>. Programmatic imports are read from
        <code> ~/.jumpstart/import.json</code>. Git credentials and tokens are stored in your
        operating system's secure keychain (macOS Keychain or Windows Credential Manager), not by
        JumpStart directly. None of this syncs to a server we control.
      </p>
      <h3>When the App uses the network</h3>
      <p>
        The App makes network requests only for things you initiate or enable, specifically:
        pushing to your own git remote, publishing a release to your git provider, checking GitHub
        for new versions of JumpStart, and, if you use the AI features, talking to a local Ollama
        model on your own machine (by default <code>http://localhost:11434</code>). AI drafting runs
        against that local model, so your prompts and project text are not sent to us or to any
        third-party AI service.
      </p>
      <h3>Optional analytics in the App</h3>
      <p>
        The App can optionally include anonymous usage analytics (Google Analytics 4 and Microsoft
        Clarity), but only if a build has been configured with analytics IDs. When enabled, these
        collect anonymous, aggregate usage signals (such as which features are opened) and never the
        contents of your projects, code, credentials, or AI prompts. If no IDs are configured, no
        analytics run in the App.
      </p>

      <h2>The website</h2>
      <p>
        The Site uses analytics to understand how people find and use JumpStart so we can improve
        it. We currently use:
      </p>
      <ul>
        <li>
          <strong>Google Analytics 4</strong> — aggregate traffic and usage (pages viewed, general
          location, device and browser type, referral source).
        </li>
        <li>
          <strong>Microsoft Clarity</strong> — anonymized interaction analytics such as heatmaps and
          session behavior to see where the page is confusing or works well.
        </li>
        <li>
          <strong>Vercel Analytics</strong> — privacy-friendly, aggregate visitor and page metrics.
        </li>
      </ul>
      <p>
        These tools may collect technical information such as your IP address (used to derive
        approximate, non-precise location and then discarded or truncated by the provider), device
        and browser details, pages visited, referring links, and on-page interactions. They may set
        cookies or similar identifiers to distinguish sessions. We use this data only in aggregate,
        to study how the Site is used and to make it better.
      </p>

      <h2>Cookies</h2>
      <p>
        The Site's analytics providers may use cookies or similar browser storage to measure usage.
        You can block or delete cookies in your browser settings; the Site remains fully functional
        without them. The App itself does not rely on advertising cookies.
      </p>

      <h2>What we do not do</h2>
      <p>
        We do not sell your personal data. We do not share it with advertisers or data brokers, and
        we do not use it to build advertising profiles. We do not upload your projects, code,
        credentials, or AI prompts from the App. The only use of any data we collect is to study and
        improve JumpStart.
      </p>

      <h2>Third-party services</h2>
      <p>
        Some features rely on third parties that have their own privacy practices, including Google
        (Analytics), Microsoft (Clarity), Vercel (hosting and analytics), and your chosen git
        provider (for example GitHub) when you push or publish. Ollama runs locally on your machine.
        We encourage you to review those providers' privacy policies for details on their handling
        of data.
      </p>

      <h2>Data retention</h2>
      <p>
        App data lives on your computer for as long as you keep it; deleting the JumpStart config
        files or uninstalling the App removes it. Site analytics data is retained by the relevant
        analytics provider according to that provider's standard retention settings.
      </p>

      <h2>Your rights</h2>
      <p>
        Depending on where you live, including under India's Digital Personal Data Protection Act,
        2023, you may have rights to access, correct, or request deletion of personal data, and to
        withdraw consent. Because the App keeps your data on your own device, you already control it
        directly. For anything relating to the Site, contact us using the details below and we will
        respond within a reasonable time.
      </p>

      <h2>Children</h2>
      <p>
        JumpStart is a developer tool intended for a general, adult audience and is not directed at
        children. We do not knowingly collect personal data from children.
      </p>

      <h2>Changes to this policy</h2>
      <p>
        We may update this Privacy Policy from time to time. When we do, we will revise the "Last
        updated" date above. Material changes will be reflected on this page.
      </p>

      <h2>Contact</h2>
      <p>
        Questions about this policy or your data can be sent to
        <a href="mailto:support@workvar.com"> support@workvar.com</a>.
      </p>
    </LegalLayout>
  );
}
