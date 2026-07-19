import { onMount } from "solid-js";
import Nav from "./components/Nav";
import Hero from "./components/Hero";
import Features from "./components/Features";
import WhatItDoes from "./components/WhatItDoes";
import Ship from "./components/Ship";
import AiBoard from "./components/AiBoard";
import Faq from "./components/Faq";
import Footer from "./components/Footer";
import { initAnimations } from "./animations";

export default function App() {
  onMount(initAnimations);
  return (
    <>
      <Nav />
      <Hero />
      <Features />
      <WhatItDoes />
      <Ship />
      <AiBoard />
      <Faq />
      <Footer />
    </>
  );
}
