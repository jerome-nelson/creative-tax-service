# **📘 Style Guide: Engineering Update Entries**

Use this guide to write consistent, professional changelog or engineering update entries that summarize completed technical work clearly and concisely.

---

## **✏️ Titles**

**Purpose:** Communicate the nature of the work in a clear, action-oriented way.

### **✅ Do**

* Use **past participle verbs**: `Completed`, `Fixed`, `Implemented`, `Patched`, etc.

* Make it **specific, but not overly detailed** (avoid component names or task IDs).

* Reflect what was done and why (e.g., "Extended UI Functionality Within Ongoing Migration Project").

### **🚫 Don’t**

* Mention sub-task status or internal identifiers (e.g., “FOR2-123” or “sub-task of…”).

* Include names of contributors or reviewers.

* Use terms like “minor update” or “small change” (stay outcome-focused).

### **💡 Examples**

* ✅ *Fixed Issue Affecting Data Synchronization*

* ✅ *Completed UI Enhancement for Ongoing Migration*

* 🚫 *FormBuilderHeaderUpdate Sub-Task*

* 🚫 *Small Cleanup by John*

---

## **📝 Paragraphs**

**Purpose:** Describe the issue or objective, how it was addressed, and any blockers or context.

### **✅ Structure**

1. **Problem or context** (e.g., issue, gap, or goal).

2. **Action taken** (e.g., implementation, fix, enhancement).

3. **Resolution status** (e.g., completed, merged, deployed).

4. **Blockers**, if relevant (and note if resolved).

5. **Reference links** at the end.

### **✅ Voice & Tone**

* Use **third-person passive voice**:

  * *"A UI enhancement had been implemented..."*

  * *"The issue had been resolved after..."*

* Keep the tone **professional, clear, and neutral**.

* Avoid technical deep-dives unless necessary for clarity.

### **🚫 Avoid**

* Names of people or teams.

* Internal component names (e.g., Form Builder, content-api).

* Overly technical descriptions.

---

## **🔗 Reference Links**

Place links at the end of each entry in a consistent format.

**Format:**

makefile  
CopyEdit  
`**Links:**`    
`https://gitlab.example.com/...`    
`https://jira.example.com/...`

---

## **🧭 General Best Practices**

* 🔄 Keep entries **generalized** (refer to *"a UI enhancement"*, not *"the change preview dropdown"*).

* 🎯 Stay **outcome-focused** (what was achieved or resolved).

* ✍️ Assume your audience includes **non-engineers** (e.g., product, leadership).

* ⏱ Keep it **short and informative**—1 title \+ 1 paragraph.

---

## **✅ Sample Entry**

### **Extended UI Functionality Within Ongoing Migration Project**

Additional UI functionality had been implemented as part of an ongoing interface project tied to a broader software migration. The work was initially blocked but was later unblocked and finalized.

**Links:**  
 [https://<gitlab.url>/<project>/-/merge\_requests/<id>](https://<gitlab.url>/<project>/-/merge\_requests/<id>)  
 [https://<jira-cloud-name>.atlassian.net/browse/<ISSUE>](https://<jira-cloud-name>.atlassian.net/browse/<ISSUE>)

---