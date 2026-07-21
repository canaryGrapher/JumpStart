import LegalLayout from "./LegalLayout";

export default function Terms() {
  return (
    <LegalLayout title="Terms of Use" updated="July 21, 2026">
      <p>
        These Terms of Use ("Terms") govern your use of the JumpStart desktop application (the
        "App") and the JumpStart website at jumpstart.workvar.com (the "Site"), together the
        "Service", provided by the JumpStart project, a project of workvar.com ("we", "us"). By
        downloading, installing, or using the Service, you agree to these Terms. If you do not
        agree, please do not use the Service.
      </p>

      <h2>License to use JumpStart</h2>
      <p>
        JumpStart is free and open source. The source code is available at
        <a href="https://github.com/canaryGrapher/JumpStart"> github.com/canaryGrapher/JumpStart</a>
        and your use of the code is governed by the license published in that repository. Subject to
        that license, you may install and use the App on your own machines for personal or
        commercial development work.
      </p>

      <h2>Your responsibilities</h2>
      <p>
        You are responsible for how you use JumpStart, including the projects, processes, commands,
        and credentials you configure it to run. JumpStart runs the commands you tell it to run and
        manages the processes you add, and you are responsible for ensuring you have the right to do
        so and that those actions are safe on your system. You are also responsible for keeping
        backups of your own data.
      </p>

      <h2>Acceptable use</h2>
      <p>
        You agree not to use JumpStart to break the law, to infringe others' rights, to run
        malicious software, or to interfere with systems you are not authorized to access. You agree
        not to misrepresent the origin of JumpStart or remove required notices from the source.
      </p>

      <h2>Your data</h2>
      <p>
        JumpStart is local-first. Your project configuration and board data are stored on your own
        machine (see the <a href="#/privacy">Privacy Policy</a>). You retain all ownership of your
        code, projects, and content. We claim no rights over them.
      </p>

      <h2>Third-party services</h2>
      <p>
        JumpStart works with third-party tools and services, including your git provider (for
        example GitHub) for pushing and publishing, and a local Ollama model for AI features. Your
        use of those services is subject to their own terms and policies, and we are not responsible
        for them.
      </p>

      <h2>Updates and availability</h2>
      <p>
        JumpStart may check for and notify you of new versions. We may change, suspend, or
        discontinue any part of the Service at any time. Because the App is distributed as software
        you run yourself, we do not guarantee continued availability of downloads, releases, or the
        Site.
      </p>

      <h2>Disclaimer of warranties</h2>
      <p>
        The Service is provided "as is" and "as available", without warranties of any kind, whether
        express or implied, including merchantability, fitness for a particular purpose, and
        non-infringement. JumpStart is distributed without notarization by Apple, and you install
        and run it at your own discretion. We do not warrant that the Service will be uninterrupted,
        error-free, or free of harmful components.
      </p>

      <h2>Limitation of liability</h2>
      <p>
        To the maximum extent permitted by law, we will not be liable for any indirect, incidental,
        special, consequential, or punitive damages, or for any loss of data, profits, or business,
        arising out of or related to your use of or inability to use the Service, even if we have
        been advised of the possibility of such damages. Because JumpStart is provided free of
        charge, our total liability for any claim relating to the Service is limited to the amount
        you paid for it, which is zero.
      </p>

      <h2>Intellectual property</h2>
      <p>
        The JumpStart name, logo, and branding are the property of their respective owners. The open
        source license grants rights to the code, but does not grant rights to use our name or logo
        in a way that implies endorsement without permission.
      </p>

      <h2>Governing law</h2>
      <p>
        These Terms are governed by the laws of India, without regard to conflict-of-law principles.
        Any disputes arising from these Terms or the Service will be subject to the exclusive
        jurisdiction of the courts located in India.
      </p>

      <h2>Changes to these Terms</h2>
      <p>
        We may update these Terms from time to time. When we do, we will revise the "Last updated"
        date above. Continued use of the Service after changes take effect constitutes acceptance of
        the revised Terms.
      </p>

      <h2>Contact</h2>
      <p>
        Questions about these Terms can be sent to
        <a href="mailto:support@workvar.com"> support@workvar.com</a>.
      </p>
    </LegalLayout>
  );
}
