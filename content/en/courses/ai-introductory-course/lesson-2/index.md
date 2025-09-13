---
title: "Asking the Right Questions (Prompting Basics)"
description: "Learn prompting basics and how to ask the right questions."
date: 2025-09-13
author: "Nikita Zinovev"
lesson:
  number: 2
  course: "AI-introductory"
---

## Why Questions Matter

Imagine you’re asking a friend for help debugging your code. If you say:

> “My program doesn’t work, fix it.”

They’ll probably stare at you blankly. But if you say:

> “I’m getting a `NullPointerException` in line 45 of my Java program when trying to access `user.getName()`. Can you
> help me figure out why?”

Now they have something concrete to work with.

Talking to AI works the same way. The quality of the answer you get depends on the quality of the question you ask. This
is called **prompting** — and in this lesson, we’ll learn how to do it right.

## What is Prompting?

A “prompt” is just the text you type when talking to AI. But not all prompts are created equal. A vague prompt gets
vague results. A clear, specific prompt (with enough context) gets useful, accurate results.

Think of prompting as giving **instructions to your new coding buddy**. The more precise you are, the better it can
help.

## Example 1: Vague vs. Specific

**Vague prompt:**

> Explain recursion.

Even with a short prompt, today’s AI will probably give you a decent explanation like _“Recursion is when a function
calls itself”_ along with a simple code example (most likely using Python). Not bad!

**Better prompt (with context):**

> Explain recursion in Java with a simple example of calculating factorial. I’m a beginner, so keep it simple.

Now the answer is likely more tailored: it may give you a factorial example in Java, walk through the steps, and explain
it in beginner-friendly language.

**Key point:** For simple questions like “Explain recursion,” the difference might feel small because the AI is
already trained to give good answers. But when your question is more **complex, specific, or multistep**, the way you
phrase it makes a **huge difference**.

For example:

- Asking “Optimize my code” might get you a vague suggestion.

- Asking “Optimize this Java loop for speed, and also explain the tradeoffs” will get you a far deeper, more useful
  response.

So yes, AI is smart — but being clear and specific is what unlocks the real power when you’re dealing with harder
programming challenges.

## Example 2: Breaking Down a Task

Instead of dumping a huge request on AI (“Write me a game in Java”), break it down:

1. **Step 1 Prompt:**

   > Help me design a simple text-based number guessing game in Java. Start by explaining the structure before writing
   code.

2. **Step 2 Prompt:**

   > Great, now write the code for the main game loop.

3. **Step 3 Prompt:**

   > Can you also add input validation so the game doesn’t crash if the user enters text instead of numbers?

Why is this approach better?

- **AI has a limited “context window.”**  
  The context window is like the AI’s short-term memory — it can only “see” a certain amount of text at once (your
  question + its own answer + previous conversation). If you throw in one giant, messy request, the important details
  might get lost. Smaller prompts keep the focus clear.

- **Focus improves quality.**  
  If you ask AI to do _one thing_ (like explain structure, or write one loop), it can give you a sharper, higher-quality
  answer. Big, vague prompts often lead to shallow or incomplete answers.

- **You stay in control.**  
  By guiding step by step, you decide the direction. Instead of getting a random version of a game you didn’t ask for,
  you shape it as you go.

Think of it like pair programming: you wouldn’t tell a colleague “build the whole app right now.” You’d plan together,
code piece by piece, and adjust as you go. Talking to AI works the same way.

## Ready-to-Use Prompts

Here are two copy-paste prompts you can try right away:

> I’m a beginner learning Java. Explain how `for` loops work using a simple code example. Pretend you’re explaining it
> to a 12-year-old.

> I have this error in Java: `Exception in thread "main" java.lang.ArrayIndexOutOfBoundsException: 5`. Can you explain
> what this error means, why it happens, and how I might fix it?

Notice how both prompts include **context** (beginner level, Java, what kind of explanation you want).

## Prompting Best Practices

Here are four golden rules for better results:

1. **Be specific** – Instead of “Explain arrays,” try “Explain arrays in Java with a short code snippet.”

2. **Give context** – Tell the AI your background (beginner, intermediate) and your goal.

3. **Break it down** – Don’t overload the AI. Smaller steps → better answers.

4. **Iterate** – Don’t expect perfection on the first try. Treat it like a conversation: refine, ask follow-ups, and
   build step by step.

**Pro-tip:** Adding phrases like _“Think step by step”_ often makes the AI slow down and give clearer, more structured
answers.

## Mini-Exercise

Open ChatGPT (or another AI tool) and try this:

1. Ask a **vague question** (like “What’s a class in Java?”).

2. Then, **rewrite the same question with more context** (e.g., “Explain Java classes with a simple example of a `Car`
   class. Keep it beginner-friendly.”).

3. Finally, try a **step-by-step prompt** (e.g., “First explain what a Java class is in plain English. Then give me an
   example with a `Car` class. Then explain each part of the code.”).

4. Compare the three answers. Which one is most useful? Why?

## Summary

Prompting is the art of **asking better questions**.

- Be specific.

- Give context.

- Break tasks into smaller steps.

- Iterate and refine.

Remember: the AI isn’t a mind reader—it’s more like a helpful coding buddy who works best when you explain what you need
clearly.

In the next lesson, we’ll see how to use these prompting skills for **debugging and troubleshooting real code problems.**

---
<br>

- [← Back to the course](/courses/ai-introductory-course/)
- [← Previous lesson](/courses/ai-introductory-course/lesson-1/)
<!-- - [Next lesson →](/courses/ai-introductory-course/lesson-3/) -->